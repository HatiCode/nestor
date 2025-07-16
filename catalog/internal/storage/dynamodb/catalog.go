package dynamodb

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/HatiCode/nestor/catalog/internal/storage"
	"github.com/HatiCode/nestor/catalog/pkg/cache"
	"github.com/HatiCode/nestor/catalog/pkg/models"
	"github.com/HatiCode/nestor/shared/pkg/json"
	"github.com/HatiCode/nestor/shared/pkg/logging"
)

type catalogStore struct {
	client    *Client
	cache     cache.Cache
	logger    logging.Logger
	tableName string
	config    *Config
}

func NewCatalogStore(config *Config, cache cache.Cache, logger logging.Logger) (storage.CatalogStore, error) {
	if config == nil {
		return nil, fmt.Errorf("DynamoDB config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid DynamoDB config: %w", err)
	}

	client, err := NewClient(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create DynamoDB client: %w", err)
	}

	store := &catalogStore{
		client:    client,
		cache:     cache,
		logger:    logger.With("component", "dynamodb_catalog_store"),
		tableName: config.GetTableName(),
		config:    config,
	}

	if config.AutoCreateTable {
		if err := store.ensureTable(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to ensure table exists: %w", err)
		}
	}

	return store, nil
}

// GetComponent retrieves a specific component by name and version
func (s *catalogStore) GetComponent(ctx context.Context, name, version string) (*models.ComponentDefinition, error) {
	s.logger.DebugContext(ctx, "getting component", "name", name, "version", version)

	if s.cache != nil {
		cacheKey := s.buildComponentCacheKey(name, version)
		if cached := s.cache.Get(ctx, cacheKey); cached != nil {
			if component, ok := cached.(*models.ComponentDefinition); ok {
				s.logger.DebugContext(ctx, "component found in cache", "name", name, "version", version)
				return component, nil
			}
		}
	}

	// Query DynamoDB
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: s.buildComponentPK(name)},
			"SK": &types.AttributeValueMemberS{Value: s.buildVersionSK(version)},
		},
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get component from DynamoDB",
			"name", name, "version", version, "error", err)
		return nil, s.wrapDynamoDBError(err, "GetComponent")
	}

	if result.Item == nil {
		return nil, storage.ErrComponentNotFound.
			WithDetail("name", name).
			WithDetail("version", version)
	}

	// Unmarshal DynamoDB item
	var dbItem ComponentItem
	if err := attributevalue.UnmarshalMap(result.Item, &dbItem); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal component",
			"name", name, "version", version, "error", err)
		return nil, fmt.Errorf("failed to unmarshal component: %w", err)
	}

	component := dbItem.ToComponentDefinition()

	// Cache the result
	if s.cache != nil {
		cacheKey := s.buildComponentCacheKey(name, version)
		if err := s.cache.Set(ctx, cacheKey, component, 5*time.Minute); err != nil {
			s.logger.WarnContext(ctx, "failed to cache component", "error", err)
		}
	}

	s.logger.DebugContext(ctx, "component retrieved from DynamoDB", "name", name, "version", version)
	return component, nil
}

// GetLatestComponent retrieves the latest version of a component
func (s *catalogStore) GetLatestComponent(ctx context.Context, name string) (*models.ComponentDefinition, error) {
	s.logger.DebugContext(ctx, "getting latest component", "name", name)

	// Check cache for latest version pointer
	var latestVersion string
	if s.cache != nil {
		cacheKey := s.buildLatestVersionCacheKey(name)
		if cached := s.cache.Get(ctx, cacheKey); cached != nil {
			if version, ok := cached.(string); ok {
				latestVersion = version
				s.logger.DebugContext(ctx, "latest version found in cache", "name", name, "version", version)
				return s.GetComponent(ctx, name, version)
			}
		}
	}

	// Query DynamoDB for the latest version
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":        &types.AttributeValueMemberS{Value: s.buildComponentPK(name)},
			":sk_prefix": &types.AttributeValueMemberS{Value: "VERSION#"},
		},
		ScanIndexForward: aws.Bool(false), // Sort descending to get latest first
		Limit:            aws.Int32(1),
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to query latest component version", "name", name, "error", err)
		return nil, s.wrapDynamoDBError(err, "GetLatestComponent")
	}

	if len(result.Items) == 0 {
		return nil, storage.ErrComponentNotFound.WithDetail("name", name)
	}

	var dbItem ComponentItem
	if err := attributevalue.UnmarshalMap(result.Items[0], &dbItem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal latest component: %w", err)
	}

	component := dbItem.ToComponentDefinition()

	// Cache the latest version pointer
	if s.cache != nil {
		cacheKey := s.buildLatestVersionCacheKey(name)
		if err := s.cache.Set(ctx, cacheKey, component.Metadata.Version, 2*time.Minute); err != nil {
			s.logger.WarnContext(ctx, "failed to cache latest version", "error", err)
		}
	}

	return component, nil
}

// ComponentExists checks if a component version exists
func (s *catalogStore) ComponentExists(ctx context.Context, name, version string) (bool, error) {
	s.logger.DebugContext(ctx, "checking component existence", "name", name, "version", version)

	// Check cache first
	if s.cache != nil {
		cacheKey := s.buildComponentCacheKey(name, version)
		if s.cache.Exists(ctx, cacheKey) {
			return true, nil
		}
	}

	// Check DynamoDB with minimal data transfer
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: s.buildComponentPK(name)},
			"SK": &types.AttributeValueMemberS{Value: s.buildVersionSK(version)},
		},
		ProjectionExpression: aws.String("PK"), // Only retrieve key to check existence
	})
	if err != nil {
		return false, s.wrapDynamoDBError(err, "ComponentExists")
	}

	return result.Item != nil, nil
}

// ListComponents retrieves components with filtering and pagination
func (s *catalogStore) ListComponents(ctx context.Context, req *storage.ListComponentsRequest) (*storage.ListComponentsResponse, error) {
	s.logger.DebugContext(ctx, "listing components", "limit", req.Limit)

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Build scan input with filters
	scanInput := s.buildScanInput(req)

	result, err := s.client.Scan(ctx, scanInput)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to scan components", "error", err)
		return nil, s.wrapDynamoDBError(err, "ListComponents")
	}

	// Unmarshal results
	components := make([]*models.ComponentDefinition, 0, len(result.Items))
	for _, item := range result.Items {
		var dbItem ComponentItem
		if err := attributevalue.UnmarshalMap(item, &dbItem); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal component", "error", err)
			continue
		}
		components = append(components, dbItem.ToComponentDefinition())
	}

	// Apply post-scan filtering and sorting
	components = s.applyPostScanFilters(components, req.Filters)
	components = s.applySorting(components, req.SortBy, req.SortOrder)

	// Handle pagination
	var nextToken string
	hasMore := false
	if result.LastEvaluatedKey != nil {
		nextToken = s.encodeToken(result.LastEvaluatedKey)
		hasMore = true
	}

	response := &storage.ListComponentsResponse{
		Components: components,
		NextToken:  nextToken,
		Total:      int64(len(components)), // This is approximate - would need separate count query for accuracy
		HasMore:    hasMore,
	}

	s.logger.DebugContext(ctx, "listed components", "count", len(components), "has_more", hasMore)
	return response, nil
}

// GetComponentVersions gets all versions of a component
func (s *catalogStore) GetComponentVersions(ctx context.Context, name string) ([]*models.ComponentVersion, error) {
	s.logger.DebugContext(ctx, "getting component versions", "name", name)

	// Query all versions for this component
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":        &types.AttributeValueMemberS{Value: s.buildComponentPK(name)},
			":sk_prefix": &types.AttributeValueMemberS{Value: "VERSION#"},
		},
		ScanIndexForward: aws.Bool(false), // Sort descending (latest first)
	})
	if err != nil {
		return nil, s.wrapDynamoDBError(err, "GetComponentVersions")
	}

	versions := make([]*models.ComponentVersion, 0, len(result.Items))
	for _, item := range result.Items {
		var dbItem ComponentItem
		if err := attributevalue.UnmarshalMap(item, &dbItem); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal component version", "error", err)
			continue
		}

		// Convert ComponentDefinition to ComponentVersion
		component := dbItem.ToComponentDefinition()
		version := &models.ComponentVersion{
			ComponentName: component.Metadata.Name,
			Version:       component.Metadata.Version,
			CreatedAt:     component.Metadata.CreatedAt,
			Status:        models.VersionStatusActive, // Simplified
		}
		versions = append(versions, version)
	}

	return versions, nil
}

// CreateComponent creates a new component version
func (s *catalogStore) CreateComponent(ctx context.Context, component *models.ComponentDefinition) error {
	s.logger.InfoContext(ctx, "creating component",
		"name", component.Metadata.Name, "version", component.Metadata.Version)

	// Validate component
	if err := component.Validate(); err != nil {
		return fmt.Errorf("invalid component: %w", err)
	}

	// Check if component already exists
	exists, err := s.ComponentExists(ctx, component.Metadata.Name, component.Metadata.Version)
	if err != nil {
		return fmt.Errorf("failed to check component existence: %w", err)
	}
	if exists {
		return storage.ErrComponentExists.
			WithDetail("name", component.Metadata.Name).
			WithDetail("version", component.Metadata.Version)
	}

	// Convert to DynamoDB item
	dbItem := NewComponentItemFromDefinition(component)
	item, err := attributevalue.MarshalMap(dbItem)
	if err != nil {
		return fmt.Errorf("failed to marshal component: %w", err)
	}

	// Create with condition to prevent overwrites
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create component",
			"name", component.Metadata.Name, "version", component.Metadata.Version, "error", err)
		return s.wrapDynamoDBError(err, "CreateComponent")
	}

	// Invalidate caches
	s.invalidateComponentCaches(ctx, component.Metadata.Name)

	s.logger.InfoContext(ctx, "component created successfully",
		"name", component.Metadata.Name, "version", component.Metadata.Version)

	return nil
}

// UpdateComponent updates an existing component
func (s *catalogStore) UpdateComponent(ctx context.Context, component *models.ComponentDefinition) error {
	s.logger.InfoContext(ctx, "updating component",
		"name", component.Metadata.Name, "version", component.Metadata.Version)

	// Validate component
	if err := component.Validate(); err != nil {
		return fmt.Errorf("invalid component: %w", err)
	}

	// Convert to DynamoDB item
	dbItem := NewComponentItemFromDefinition(component)
	item, err := attributevalue.MarshalMap(dbItem)
	if err != nil {
		return fmt.Errorf("failed to marshal component: %w", err)
	}

	// Update with condition that item must exist
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update component",
			"name", component.Metadata.Name, "version", component.Metadata.Version, "error", err)
		return s.wrapDynamoDBError(err, "UpdateComponent")
	}

	// Invalidate caches
	s.invalidateComponentCaches(ctx, component.Metadata.Name)

	s.logger.InfoContext(ctx, "component updated successfully",
		"name", component.Metadata.Name, "version", component.Metadata.Version)

	return nil
}

// DeleteComponent deletes a component version
func (s *catalogStore) DeleteComponent(ctx context.Context, name, version string, reason string) error {
	s.logger.InfoContext(ctx, "deleting component", "name", name, "version", version, "reason", reason)

	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: s.buildComponentPK(name)},
			"SK": &types.AttributeValueMemberS{Value: s.buildVersionSK(version)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete component", "name", name, "version", version, "error", err)
		return s.wrapDynamoDBError(err, "DeleteComponent")
	}

	// Invalidate caches
	s.invalidateComponentCaches(ctx, name)

	s.logger.InfoContext(ctx, "component deleted successfully", "name", name, "version", version)
	return nil
}

// DeleteComponentVersion is an alias for DeleteComponent for compatibility
func (s *catalogStore) DeleteComponentVersion(ctx context.Context, name, version string, reason string) error {
	return s.DeleteComponent(ctx, name, version, reason)
}

// BatchCreateComponents creates multiple components efficiently
func (s *catalogStore) BatchCreateComponents(ctx context.Context, components []*models.ComponentDefinition) error {
	if len(components) == 0 {
		return nil
	}

	s.logger.InfoContext(ctx, "batch creating components", "count", len(components))

	// Validate all components first
	for _, component := range components {
		if err := component.Validate(); err != nil {
			return fmt.Errorf("invalid component %s:%s: %w",
				component.Metadata.Name, component.Metadata.Version, err)
		}
	}

	// Process in batches (DynamoDB limit is 25 items per batch)
	const batchSize = 25
	for i := 0; i < len(components); i += batchSize {
		end := i + batchSize
		if end > len(components) {
			end = len(components)
		}

		batch := components[i:end]
		if err := s.batchCreateComponentsBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to create batch %d-%d: %w", i, end, err)
		}
	}

	// Invalidate caches for all affected components
	componentNames := make(map[string]bool)
	for _, component := range components {
		componentNames[component.Metadata.Name] = true
	}
	for name := range componentNames {
		s.invalidateComponentCaches(ctx, name)
	}

	s.logger.InfoContext(ctx, "batch created components successfully", "count", len(components))
	return nil
}

func (s *catalogStore) BatchGetComponents(ctx context.Context, components []storage.ComponentRef) ([]*models.ComponentDefinition, error) {
	if len(components) == 0 {
		return nil, nil
	}

	return nil, nil
}

// SearchComponents performs component search
func (s *catalogStore) SearchComponents(ctx context.Context, req *storage.SearchComponentsRequest) (*storage.SearchComponentsResponse, error) {
	s.logger.DebugContext(ctx, "searching components", "query", req.Query)

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid search request: %w", err)
	}

	start := time.Now()

	// For simplicity, we'll use scan with filter expressions
	scanInput := s.buildSearchScanInput(req)

	result, err := s.client.Scan(ctx, scanInput)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to search components", "error", err)
		return nil, s.wrapDynamoDBError(err, "SearchComponents")
	}

	// Unmarshal and filter results
	components := make([]*models.ComponentDefinition, 0, len(result.Items))
	for _, item := range result.Items {
		var dbItem ComponentItem
		if err := attributevalue.UnmarshalMap(item, &dbItem); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal component in search", "error", err)
			continue
		}

		component := dbItem.ToComponentDefinition()
		if s.matchesSearchQuery(component, req.Query) {
			components = append(components, component)
		}
	}

	queryTime := time.Since(start)

	// Handle pagination
	var nextToken string
	if result.LastEvaluatedKey != nil {
		nextToken = s.encodeToken(result.LastEvaluatedKey)
	}

	response := &storage.SearchComponentsResponse{
		Components: components,
		NextToken:  nextToken,
		Total:      int64(len(components)),
		QueryTime:  queryTime,
		Facets:     make(map[string]*storage.FacetResult), // TODO: Implement faceting
	}

	s.logger.DebugContext(ctx, "search completed",
		"query", req.Query, "results", len(components), "query_time_ms", queryTime.Milliseconds())

	return response, nil
}

// FindComponentsByProvider finds components by provider
func (s *catalogStore) FindComponentsByProvider(ctx context.Context, provider string) ([]*models.ComponentDefinition, error) {
	s.logger.DebugContext(ctx, "finding components by provider", "provider", provider)

	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		FilterExpression: aws.String("Provider = :provider"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":provider": &types.AttributeValueMemberS{Value: provider},
		},
	}

	result, err := s.client.Scan(ctx, scanInput)
	if err != nil {
		return nil, s.wrapDynamoDBError(err, "FindComponentsByProvider")
	}

	components := make([]*models.ComponentDefinition, 0, len(result.Items))
	for _, item := range result.Items {
		var dbItem ComponentItem
		if err := attributevalue.UnmarshalMap(item, &dbItem); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal component", "error", err)
			continue
		}
		components = append(components, dbItem.ToComponentDefinition())
	}

	return components, nil
}

// FindComponentsByCategory finds components by category
func (s *catalogStore) FindComponentsByCategory(ctx context.Context, category string) ([]*models.ComponentDefinition, error) {
	s.logger.DebugContext(ctx, "finding components by category", "category", category)

	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		FilterExpression: aws.String("Category = :category"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":category": &types.AttributeValueMemberS{Value: category},
		},
	}

	result, err := s.client.Scan(ctx, scanInput)
	if err != nil {
		return nil, s.wrapDynamoDBError(err, "FindComponentsByCategory")
	}

	components := make([]*models.ComponentDefinition, 0, len(result.Items))
	for _, item := range result.Items {
		var dbItem ComponentItem
		if err := attributevalue.UnmarshalMap(item, &dbItem); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal component", "error", err)
			continue
		}
		components = append(components, dbItem.ToComponentDefinition())
	}

	return components, nil
}

// FindComponentsByLabels finds components by labels
func (s *catalogStore) FindComponentsByLabels(ctx context.Context, labels map[string]string) ([]*models.ComponentDefinition, error) {
	s.logger.DebugContext(ctx, "finding components by labels", "label_count", len(labels))

	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	}

	result, err := s.client.Scan(ctx, scanInput)
	if err != nil {
		return nil, s.wrapDynamoDBError(err, "FindComponentsByLabels")
	}

	components := make([]*models.ComponentDefinition, 0)
	for _, item := range result.Items {
		var dbItem ComponentItem
		if err := attributevalue.UnmarshalMap(item, &dbItem); err != nil {
			continue
		}

		component := dbItem.ToComponentDefinition()
		if s.matchesLabels(component, labels) {
			components = append(components, component)
		}
	}

	return components, nil
}

// FindDependents finds components that depend on the given component
func (s *catalogStore) FindDependents(ctx context.Context, componentName string) ([]*models.ComponentDefinition, error) {
	s.logger.DebugContext(ctx, "finding dependents", "component", componentName)

	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	}

	result, err := s.client.Scan(ctx, scanInput)
	if err != nil {
		return nil, s.wrapDynamoDBError(err, "FindDependents")
	}

	dependents := make([]*models.ComponentDefinition, 0)
	for _, item := range result.Items {
		var dbItem ComponentItem
		if err := attributevalue.UnmarshalMap(item, &dbItem); err != nil {
			continue
		}

		component := dbItem.ToComponentDefinition()
		if s.hasDependencyOn(component, componentName) {
			dependents = append(dependents, component)
		}
	}

	return dependents, nil
}

// FindDependencies finds dependencies of a component
func (s *catalogStore) FindDependencies(ctx context.Context, componentName, version string, recursive bool) ([]*models.ComponentDefinition, error) {
	s.logger.DebugContext(ctx, "finding dependencies",
		"component", componentName, "version", version, "recursive", recursive)

	// Get the component first
	component, err := s.GetComponent(ctx, componentName, version)
	if err != nil {
		return nil, err
	}

	// Get direct dependencies
	dependencies := make([]*models.ComponentDefinition, 0)
	seen := make(map[string]bool)

	for _, dep := range component.Spec.Dependencies {
		depName := dep.Name
		depVersion := dep.Version // Simplified - should resolve version constraint

		if seen[depName] {
			continue
		}
		seen[depName] = true

		depComponent, err := s.GetLatestComponent(ctx, depName)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to get dependency", "dependency", depName, "error", err)
			continue
		}

		dependencies = append(dependencies, depComponent)

		// Recursive dependencies
		if recursive {
			subDeps, err := s.FindDependencies(ctx, depName, depComponent.Metadata.Version, true)
			if err != nil {
				s.logger.WarnContext(ctx, "failed to get recursive dependencies", "dependency", depName, "error", err)
				continue
			}

			for _, subDep := range subDeps {
				if !seen[subDep.Metadata.Name] {
					seen[subDep.Metadata.Name] = true
					dependencies = append(dependencies, subDep)
				}
			}
		}
	}

	return dependencies, nil
}

// Placeholder implementations for versioning operations
func (s *catalogStore) CreateComponentVersion(ctx context.Context, version *models.ComponentVersion) error {
	return fmt.Errorf("CreateComponentVersion not yet implemented")
}

func (s *catalogStore) GetComponentVersion(ctx context.Context, name, version string) (*models.ComponentVersion, error) {
	return nil, fmt.Errorf("GetComponentVersion not yet implemented")
}

func (s *catalogStore) GetLatestMajorVersion(ctx context.Context, name string, majorVersion int) (*models.ComponentVersion, error) {
	return nil, fmt.Errorf("GetLatestMajorVersion not yet implemented")
}

func (s *catalogStore) ListComponentChanges(ctx context.Context, name string, since time.Time) ([]*models.ComponentChange, error) {
	return nil, fmt.Errorf("ListComponentChanges not yet implemented")
}

func (s *catalogStore) CompareVersions(ctx context.Context, name, fromVersion, toVersion string) (*models.VersionDiff, error) {
	return nil, fmt.Errorf("CompareVersions not yet implemented")
}

func (s *catalogStore) FindCompatibleVersions(ctx context.Context, name, constraint string) ([]*models.ComponentVersion, error) {
	return nil, fmt.Errorf("FindCompatibleVersions not yet implemented")
}

func (s *catalogStore) GetVersionsByCommit(ctx context.Context, commitSHA string) ([]*models.ComponentVersion, error) {
	return nil, fmt.Errorf("GetVersionsByCommit not yet implemented")
}

// Health Operations

// HealthCheck verifies the store is healthy
func (s *catalogStore) HealthCheck(ctx context.Context) error {
	return s.Ping(ctx)
}

// Ping checks connectivity to DynamoDB
func (s *catalogStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}

// Helper methods

// buildComponentPK builds the partition key for a component
func (s *catalogStore) buildComponentPK(name string) string {
	return fmt.Sprintf("COMPONENT#%s", name)
}

// buildVersionSK builds the sort key for a version
func (s *catalogStore) buildVersionSK(version string) string {
	return fmt.Sprintf("VERSION#%s", version)
}

// buildComponentCacheKey builds cache key for a component
func (s *catalogStore) buildComponentCacheKey(name, version string) string {
	return fmt.Sprintf("component:%s:%s", name, version)
}

// buildLatestVersionCacheKey builds cache key for latest version pointer
func (s *catalogStore) buildLatestVersionCacheKey(name string) string {
	return fmt.Sprintf("latest:%s", name)
}

// invalidateComponentCaches invalidates all caches related to a component
func (s *catalogStore) invalidateComponentCaches(ctx context.Context, name string) {
	if s.cache == nil {
		return
	}

	// Invalidate latest version cache
	latestKey := s.buildLatestVersionCacheKey(name)
	if err := s.cache.Delete(ctx, latestKey); err != nil {
		s.logger.WarnContext(ctx, "failed to invalidate latest version cache", "name", name, "error", err)
	}

	// Note: We don't invalidate individual version caches since they're immutable
	// Only the "latest" pointer changes
}

// encodeToken encodes DynamoDB LastEvaluatedKey for pagination
func (s *catalogStore) encodeToken(lastKey map[string]types.AttributeValue) string {
	if len(lastKey) == 0 {
		return ""
	}

	// Marshal the key to JSON and base64 encode it
	keyBytes, err := json.ToJSON(lastKey)
	if err != nil {
		s.logger.Warn("failed to encode pagination token", "error", err)
		return ""
	}

	return base64.URLEncoding.EncodeToString(keyBytes)
}

// decodeToken decodes pagination token back to DynamoDB key
func (s *catalogStore) decodeToken(token string) (map[string]types.AttributeValue, error) {
	if token == "" {
		return nil, nil
	}

	keyBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination token: %w", err)
	}

	var lastKey map[string]types.AttributeValue
	if err := json.FromJSON(keyBytes, &lastKey); err != nil {
		return nil, fmt.Errorf("failed to decode pagination token: %w", err)
	}

	return lastKey, nil
}

// buildScanInput builds a DynamoDB scan input with filters
func (s *catalogStore) buildScanInput(req *storage.ListComponentsRequest) *dynamodb.ScanInput {
	input := &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	}

	if req.Limit > 0 {
		input.Limit = aws.Int32(req.Limit)
	}

	// Handle pagination
	if req.NextToken != "" {
		if lastKey, err := s.decodeToken(req.NextToken); err == nil && lastKey != nil {
			input.ExclusiveStartKey = lastKey
		}
	}

	// Build filter expressions
	var filterParts []string
	var expressionValues map[string]types.AttributeValue
	var expressionNames map[string]string

	if req.Filters != nil {
		expressionValues = make(map[string]types.AttributeValue)
		expressionNames = make(map[string]string)

		// Provider filter
		if len(req.Filters.Providers) > 0 {
			var providerParts []string
			for i, provider := range req.Filters.Providers {
				key := fmt.Sprintf(":provider%d", i)
				providerParts = append(providerParts, fmt.Sprintf("Provider = %s", key))
				expressionValues[key] = &types.AttributeValueMemberS{Value: provider}
			}
			if len(providerParts) > 0 {
				filterParts = append(filterParts, fmt.Sprintf("(%s)", strings.Join(providerParts, " OR ")))
			}
		}

		// Category filter
		if len(req.Filters.Categories) > 0 {
			var categoryParts []string
			for i, category := range req.Filters.Categories {
				key := fmt.Sprintf(":category%d", i)
				categoryParts = append(categoryParts, fmt.Sprintf("Category = %s", key))
				expressionValues[key] = &types.AttributeValueMemberS{Value: category}
			}
			if len(categoryParts) > 0 {
				filterParts = append(filterParts, fmt.Sprintf("(%s)", strings.Join(categoryParts, " OR ")))
			}
		}

		// Maturity filter
		if len(req.Filters.Maturity) > 0 {
			var maturityParts []string
			for i, maturity := range req.Filters.Maturity {
				key := fmt.Sprintf(":maturity%d", i)
				maturityParts = append(maturityParts, fmt.Sprintf("Maturity = %s", key))
				expressionValues[key] = &types.AttributeValueMemberS{Value: string(maturity)}
			}
			if len(maturityParts) > 0 {
				filterParts = append(filterParts, fmt.Sprintf("(%s)", strings.Join(maturityParts, " OR ")))
			}
		}

		// Date filters
		if req.Filters.CreatedAfter != nil {
			filterParts = append(filterParts, "CreatedAt > :created_after")
			expressionValues[":created_after"] = &types.AttributeValueMemberS{
				Value: req.Filters.CreatedAfter.Format(time.RFC3339),
			}
		}

		if req.Filters.CreatedBefore != nil {
			filterParts = append(filterParts, "CreatedAt < :created_before")
			expressionValues[":created_before"] = &types.AttributeValueMemberS{
				Value: req.Filters.CreatedBefore.Format(time.RFC3339),
			}
		}
	}

	// Apply filter expression
	if len(filterParts) > 0 {
		input.FilterExpression = aws.String(strings.Join(filterParts, " AND "))
		input.ExpressionAttributeValues = expressionValues
		if len(expressionNames) > 0 {
			input.ExpressionAttributeNames = expressionNames
		}
	}

	return input
}

// buildSearchScanInput builds scan input for search operations
func (s *catalogStore) buildSearchScanInput(req *storage.SearchComponentsRequest) *dynamodb.ScanInput {
	input := &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	}

	if req.Limit > 0 {
		input.Limit = aws.Int32(req.Limit)
	}

	if req.NextToken != "" {
		if lastKey, err := s.decodeToken(req.NextToken); err == nil && lastKey != nil {
			input.ExclusiveStartKey = lastKey
		}
	}

	return input
}

// applyPostScanFilters applies filters that couldn't be done at the DynamoDB level
func (s *catalogStore) applyPostScanFilters(components []*models.ComponentDefinition, filters *storage.ComponentFilters) []*models.ComponentDefinition {
	if filters == nil {
		return components
	}

	filtered := make([]*models.ComponentDefinition, 0, len(components))

	for _, component := range components {
		// Label filtering
		if len(filters.Labels) > 0 && !s.matchesLabels(component, filters.Labels) {
			continue
		}

		// Deployment engine filtering
		if len(filters.DeploymentEngines) > 0 {
			hasEngine := false
			for _, engine := range filters.DeploymentEngines {
				if component.SupportsEngine(engine) {
					hasEngine = true
					break
				}
			}
			if !hasEngine {
				continue
			}
		}

		filtered = append(filtered, component)
	}

	return filtered
}

// applySorting sorts components based on the request
func (s *catalogStore) applySorting(components []*models.ComponentDefinition, sortBy storage.SortField, sortOrder storage.SortOrder) []*models.ComponentDefinition {
	if sortBy == "" {
		return components
	}

	sort.Slice(components, func(i, j int) bool {
		var less bool

		switch sortBy {
		case storage.SortByName:
			less = components[i].Metadata.Name < components[j].Metadata.Name
		case storage.SortByProvider:
			less = components[i].Metadata.Provider < components[j].Metadata.Provider
		case storage.SortByCategory:
			less = components[i].Metadata.Category < components[j].Metadata.Category
		case storage.SortByCreated:
			less = components[i].Metadata.CreatedAt.Before(components[j].Metadata.CreatedAt)
		case storage.SortByUpdated:
			less = components[i].Metadata.UpdatedAt.Before(components[j].Metadata.UpdatedAt)
		case storage.SortByVersion:
			// Simple string comparison - in production, should use semantic version comparison
			less = components[i].Metadata.Version < components[j].Metadata.Version
		case storage.SortByMaturity:
			less = components[i].Metadata.Maturity < components[j].Metadata.Maturity
		default:
			return false
		}

		if sortOrder == storage.SortDesc {
			return !less
		}
		return less
	})

	return components
}

// matchesSearchQuery checks if a component matches the search query
func (s *catalogStore) matchesSearchQuery(component *models.ComponentDefinition, query string) bool {
	if query == "" {
		return true
	}

	query = strings.ToLower(query)

	// Search in name, description, provider, category
	searchFields := []string{
		strings.ToLower(component.Metadata.Name),
		strings.ToLower(component.Metadata.DisplayName),
		strings.ToLower(component.Metadata.Description),
		strings.ToLower(component.Metadata.Provider),
		strings.ToLower(component.Metadata.Category),
		strings.ToLower(component.Metadata.ResourceType),
	}

	for _, field := range searchFields {
		if strings.Contains(field, query) {
			return true
		}
	}

	// Search in labels
	for key, value := range component.Metadata.Labels {
		if strings.Contains(strings.ToLower(key), query) ||
			strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}

	return false
}

// matchesLabels checks if a component matches all provided labels
func (s *catalogStore) matchesLabels(component *models.ComponentDefinition, labels map[string]string) bool {
	if len(labels) == 0 {
		return true
	}

	if component.Metadata.Labels == nil {
		return false
	}

	for key, value := range labels {
		if componentValue, exists := component.Metadata.Labels[key]; !exists || componentValue != value {
			return false
		}
	}

	return true
}

// hasDependencyOn checks if a component depends on another component
func (s *catalogStore) hasDependencyOn(component *models.ComponentDefinition, dependencyName string) bool {
	for _, dep := range component.Spec.Dependencies {
		if dep.Name == dependencyName {
			return true
		}
	}
	return false
}

// batchCreateComponentsBatch creates a batch of components
func (s *catalogStore) batchCreateComponentsBatch(ctx context.Context, components []*models.ComponentDefinition) error {
	if len(components) == 0 {
		return nil
	}

	writeRequests := make([]types.WriteRequest, 0, len(components))

	for _, component := range components {
		dbItem := NewComponentItemFromDefinition(component)
		item, err := attributevalue.MarshalMap(dbItem)
		if err != nil {
			return fmt.Errorf("failed to marshal component %s:%s: %w",
				component.Metadata.Name, component.Metadata.Version, err)
		}

		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: item,
			},
		})
	}

	// Execute batch write
	_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			s.tableName: writeRequests,
		},
	})

	return err
}

func (s *catalogStore) batchGetComponentsBatch(ctx context.Context, refs []storage.ComponentRef) ([]*models.ComponentDefinition, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	keys := make([]map[string]types.AttributeValue, 0, len(refs))
	for _, ref := range refs {
		keys = append(keys, map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: s.buildComponentPK(ref.Name)},
			"SK": &types.AttributeValueMemberS{Value: s.buildVersionSK(ref.Version)},
		})
	}

	result, err := s.client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			s.tableName: {
				Keys: keys,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	components := make([]*models.ComponentDefinition, 0, len(result.Responses[s.tableName]))
	for _, item := range result.Responses[s.tableName] {
		var dbItem ComponentItem
		if err := attributevalue.UnmarshalMap(item, &dbItem); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal component in batch", "error", err)
			continue
		}
		components = append(components, dbItem.ToComponentDefinition())
	}

	return components, nil
}

func (s *catalogStore) wrapDynamoDBError(err error, operation string) error {
	if err == nil {
		return nil
	}

	var conditionErr *types.ConditionalCheckFailedException
	if aws.As(err, &conditionErr) {
		return storage.ErrComponentExists.WithDetail("operation", operation)
	}

	var resourceNotFoundErr *types.ResourceNotFoundException
	if aws.As(err, &resourceNotFoundErr) {
		return storage.ErrComponentNotFound.WithDetail("operation", operation)
	}

	var throttleErr *types.ProvisionedThroughputExceededException
	if aws.As(err, &throttleErr) {
		return fmt.Errorf("DynamoDB throttled during %s: %w", operation, err)
	}

	return fmt.Errorf("DynamoDB error during %s: %w", operation, err)
}

func (s *catalogStore) ensureTable(ctx context.Context) error {
	exists, err := s.client.TableExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if table exists: %w", err)
	}

	if exists {
		s.logger.Info("DynamoDB table already exists", "table", s.tableName)
		return nil
	}

	s.logger.Info("Creating DynamoDB table", "table", s.tableName)

	_, err = s.client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(s.tableName),
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("PK"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("SK"),
				KeyType:       types.KeyTypeRange,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("PK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("SK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("Provider"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("Category"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		BillingMode: types.BillingModeProvisioned,
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(int64(s.config.ReadCapacity)),
			WriteCapacityUnits: aws.Int64(int64(s.config.WriteCapacity)),
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("ProviderIndex"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("Provider"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("SK"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(int64(s.config.ReadCapacity)),
					WriteCapacityUnits: aws.Int64(int64(s.config.WriteCapacity)),
				},
			},
			{
				IndexName: aws.String("CategoryIndex"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("Category"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("SK"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(int64(s.config.ReadCapacity)),
					WriteCapacityUnits: aws.Int64(int64(s.config.WriteCapacity)),
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	s.logger.Info("Waiting for table to be active", "table", s.tableName)
	return s.client.WaitForTable(ctx)
}
