package dynamodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/HatiCode/nestor/catalog/internal/storage"
	"github.com/HatiCode/nestor/catalog/pkg/models"
	"github.com/HatiCode/nestor/shared/pkg/logging"
)

// catalogStore implements the storage.CatalogStore interface using DynamoDB
// Following Single Responsibility Principle - focused only on DynamoDB storage operations
type catalogStore struct {
	client    *Client
	cache     storage.Cache
	logger    logging.Logger
	tableName string
	config    *Config
}

// NewCatalogStore creates a new DynamoDB-backed catalog store
// Uses the existing config.go structure and follows Dependency Inversion Principle
func NewCatalogStore(config *Config, cache storage.Cache, logger logging.Logger) (storage.CatalogStore, error) {
	if config == nil {
		return nil, fmt.Errorf("DynamoDB config is required")
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid DynamoDB config: %w", err)
	}

	// Create DynamoDB client
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

	// Create table if auto-create is enabled
	if config.AutoCreateTable {
		if err := store.ensureTable(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to ensure table exists: %w", err)
		}
	}

	return store, nil
}

// Component Reader Operations

// GetComponent retrieves a specific component version
func (s *catalogStore) GetComponent(ctx context.Context, name, version string) (*models.ComponentDefinition, error) {
	s.logger.DebugContext(ctx, "getting component", "name", name, "version", version)

	// Check cache first
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
		Limit:           aws.Int32(1),
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

// BatchGetComponents retrieves multiple components efficiently
func (s *catalogStore) BatchGetComponents(ctx context.Context, refs []storage.ComponentRef) ([]*models.ComponentDefinition, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	s.logger.DebugContext(ctx, "batch getting components", "count", len(refs))

	// DynamoDB batch get has a limit of 100 items
	const batchSize = 100
	var allComponents []*models.ComponentDefinition

	for i := 0; i < len(refs); i += batchSize {
		end := i + batchSize
		if end > len(refs) {
			end = len(refs)
		}

		batch := refs[i:end]
		components, err := s.batchGetComponentsBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to get batch %d-%d: %w", i, end, err)
		}

		allComponents = append(allComponents, components...)
	}

	s.logger.DebugContext(ctx, "batch got components", "requested", len(refs), "retrieved", len(allComponents))
	return allComponents, nil
}

// Component Writer Operations

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
func (s *catalogStore) DeleteComponent(ctx context.Context, name, version string) error {
	s.logger.InfoContext(ctx, "deleting component", "name", name, "version", version)

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

// Component Search Operations

// SearchComponents performs component search
func (s *catalogStore) SearchComponents(ctx context.Context, req *storage.SearchComponentsRequest) (*storage.SearchComponentsResponse, error) {
	s.logger.DebugContext(ctx, "searching components", "query", req.Query)

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid search request: %w", err)
	}

	start := time.Now()

	// For simplicity, we'll use scan with filter expressions
	// In production, you might want to use DynamoDB search capabilities or external search
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
	hasMore := false
	if result.LastEvaluatedKey != nil {
		nextToken = s.encodeToken(result.LastEvaluatedKey)
		hasMore = true
	}

	response := &storage.SearchComponentsResponse{
		Components: components,
		NextToken:  nextToken,
		Total:      int64(len(components)),
		HasMore:    hasMore,
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

	// Use GSI if available, otherwise scan with filter
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

	// This is simplified - in production you might want a more efficient approach
	// using GSI or denormalized label data
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

	// This requires scanning and checking dependencies - could be optimized with GSI
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
		// Parse dependency name and version
		depName := dep.Name
		depVersion := dep.Version // This might be a constraint, would need resolution

		if seen[depName] {
			continue
		}
		seen[depName] = true

		depComponent, err := s.GetLatestComponent(ctx, depName) // Simplified - should resolve version constraint
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

// Component Versioning Operations - placeholder implementations

// CreateComponentVersion creates a component version record
func (s *catalogStore) CreateComponentVersion(ctx context.Context, version *models.ComponentVersion) error {
	// TODO: Implement version tracking
	return fmt.Errorf("CreateComponentVersion not yet implemented")
}

// GetComponentVersion gets a specific component version record
func (s *catalogStore) GetComponentVersion(ctx context.Context, name, version string) (*models.ComponentVersion, error) {
	// TODO: Implement version tracking
	return nil, fmt.Errorf("GetComponentVersion not yet implemented")
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

// Additional versioning methods would be implemented here...
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
	return