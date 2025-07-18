package dynamodb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/HatiCode/nestor/catalog/internal/storage"
	"github.com/HatiCode/nestor/catalog/pkg/cache"
	"github.com/HatiCode/nestor/catalog/pkg/models"
	"github.com/HatiCode/nestor/shared/pkg/logging"
)

// componentStore implements the ComponentStore interface for DynamoDB
type componentStore struct {
	client    *Client
	cache     cache.Cache
	logger    logging.Logger
	tableName string
	config    *Config
}

// NewComponentStore creates a new DynamoDB-backed ComponentStore
func NewComponentStore(config *storage.StorageConfig, cache cache.Cache, logger logging.Logger) (storage.ComponentStore, error) {
	if config.DynamoDB == nil {
		return nil, storage.NewConfigurationError("DynamoDB", "DynamoDB config is required")
	}

	// Convert storage config to DynamoDB config
	dynamoConfig, err := convertStorageConfig(config.DynamoDB)
	if err != nil {
		return nil, fmt.Errorf("failed to convert storage config: %w", err)
	}

	if err := dynamoConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid DynamoDB config: %w", err)
	}

	client, err := NewClient(dynamoConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create DynamoDB client: %w", err)
	}

	store := &componentStore{
		client:    client,
		cache:     cache,
		logger:    logger.With("component", "dynamodb_component_store"),
		tableName: dynamoConfig.GetTableName(),
		config:    dynamoConfig,
	}

	if dynamoConfig.AutoCreateTable {
		if err := store.ensureTable(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to ensure table exists: %w", err)
		}
	}

	return store, nil
}

// GetComponent retrieves a specific component by name and version
func (s *componentStore) GetComponent(ctx context.Context, name, version string) (*models.ComponentDefinition, error) {
	if name == "" {
		return nil, storage.NewValidationError("name", "component name is required")
	}
	if version == "" {
		return nil, storage.NewValidationError("version", "component version is required")
	}

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
		return nil, s.wrapDynamoDBError(err, "GetComponent", name, version)
	}

	if result.Item == nil {
		return nil, storage.NewComponentNotFoundError(name, version).
			WithDetail("operation", "GetComponent")
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

// ListComponents retrieves components with filtering and pagination
func (s *componentStore) ListComponents(ctx context.Context, filters storage.ComponentFilters, pagination storage.Pagination) (*storage.ComponentList, error) {
	s.logger.DebugContext(ctx, "listing components", "limit", pagination.Limit)

	// Validate inputs
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("invalid filters: %w", err)
	}
	if err := pagination.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination: %w", err)
	}

	// Build scan input with filters
	scanInput := s.buildScanInput(&filters, &pagination)

	result, err := s.client.Scan(ctx, scanInput)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to scan components", "error", err)
		return nil, s.wrapDynamoDBError(err, "ListComponents", "", "")
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
	components = s.applyPostScanFilters(components, &filters)
	components = s.applySorting(components, pagination.SortBy, pagination.SortOrder)

	// Handle pagination
	var nextToken string
	hasMore := false
	if result.LastEvaluatedKey != nil {
		nextToken = s.encodeToken(result.LastEvaluatedKey)
		hasMore = true
	}

	response := &storage.ComponentList{
		Components: components,
		NextToken:  nextToken,
		Total:      int64(len(components)), // This is approximate - would need separate count query for accuracy
		HasMore:    hasMore,
	}

	s.logger.DebugContext(ctx, "listed components", "count", len(components), "has_more", hasMore)
	return response, nil
}

// StoreComponent stores a component definition
func (s *componentStore) StoreComponent(ctx context.Context, component *models.ComponentDefinition) error {
	if component == nil {
		return storage.NewValidationError("component", "component is required")
	}

	s.logger.InfoContext(ctx, "storing component",
		"name", component.Metadata.Name, "version", component.Metadata.Version)

	// Validate component
	if err := component.Validate(); err != nil {
		return storage.NewValidationError("component", fmt.Sprintf("invalid component: %v", err))
	}

	// Convert to DynamoDB item
	dbItem := NewComponentItemFromDefinition(component)
	item, err := attributevalue.MarshalMap(dbItem)
	if err != nil {
		return fmt.Errorf("failed to marshal component: %w", err)
	}

	// Store with upsert behavior (overwrite if exists)
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to store component",
			"name", component.Metadata.Name, "version", component.Metadata.Version, "error", err)
		return s.wrapDynamoDBError(err, "StoreComponent", component.Metadata.Name, component.Metadata.Version)
	}

	// Invalidate caches
	s.invalidateComponentCaches(ctx, component.Metadata.Name)

	s.logger.InfoContext(ctx, "component stored successfully",
		"name", component.Metadata.Name, "version", component.Metadata.Version)

	return nil
}

// GetVersionHistory gets all versions of a component
func (s *componentStore) GetVersionHistory(ctx context.Context, name string) ([]models.ComponentVersion, error) {
	if name == "" {
		return nil, storage.NewValidationError("name", "component name is required")
	}

	s.logger.DebugContext(ctx, "getting component version history", "name", name)

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
		s.logger.ErrorContext(ctx, "failed to get version history", "name", name, "error", err)
		return nil, s.wrapDynamoDBError(err, "GetVersionHistory", name, "")
	}

	versions := make([]models.ComponentVersion, 0, len(result.Items))
	for _, item := range result.Items {
		var dbItem ComponentItem
		if err := attributevalue.UnmarshalMap(item, &dbItem); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal component version", "error", err)
			continue
		}

		// Convert ComponentDefinition to ComponentVersion
		component := dbItem.ToComponentDefinition()
		version := models.ComponentVersion{
			ComponentName: component.Metadata.Name,
			Version:       component.Metadata.Version,
			CreatedAt:     component.Metadata.CreatedAt,
			Status:        models.VersionStatusActive, // Simplified for MVP
		}
		versions = append(versions, version)
	}

	s.logger.DebugContext(ctx, "retrieved version history", "name", name, "count", len(versions))
	return versions, nil
}

// HealthCheck verifies the store is healthy
func (s *componentStore) HealthCheck(ctx context.Context) error {
	s.logger.DebugContext(ctx, "performing health check")

	if err := s.client.Ping(ctx); err != nil {
		s.logger.ErrorContext(ctx, "health check failed", "error", err)
		return storage.NewStorageUnavailableError(err.Error()).
			WithDetail("operation", "HealthCheck")
	}

	s.logger.DebugContext(ctx, "health check passed")
	return nil
}

// Helper methods

func (s *componentStore) buildComponentPK(name string) string {
	return fmt.Sprintf("COMPONENT#%s", name)
}

func (s *componentStore) buildVersionSK(version string) string {
	return fmt.Sprintf("VERSION#%s", version)
}

func (s *componentStore) buildComponentCacheKey(name, version string) string {
	return fmt.Sprintf("component:%s:%s", name, version)
}

func (s *componentStore) invalidateComponentCaches(ctx context.Context, name string) {
	if s.cache == nil {
		return
	}

	// Invalidate latest version cache
	latestKey := fmt.Sprintf("latest:%s", name)
	if err := s.cache.Delete(ctx, latestKey); err != nil {
		s.logger.WarnContext(ctx, "failed to invalidate latest version cache", "name", name, "error", err)
	}
}

func (s *componentStore) wrapDynamoDBError(err error, operation string, params ...string) error {
	// Extract optional parameters for more context
	var name, version string
	if len(params) > 0 {
		name = params[0]
	}
	if len(params) > 1 {
		version = params[1]
	}

	// Convert AWS DynamoDB errors to storage errors using type assertions
	var resourceNotFoundErr *types.ResourceNotFoundException
	var conditionalCheckFailedErr *types.ConditionalCheckFailedException
	var throttledErr *types.ProvisionedThroughputExceededException

	switch {
	case errors.As(err, &resourceNotFoundErr):
		// Resource not found error
		if name != "" {
			return storage.NewComponentNotFoundError(name, version).
				WithDetail("operation", operation)
		}
		return storage.NewResourceNotFoundError("component", "unknown").
			WithDetail("operation", operation)

	case errors.As(err, &conditionalCheckFailedErr):
		// Conditional check failed (resource exists)
		if name != "" {
			return storage.NewComponentExistsError(name, version).
				WithDetail("operation", operation)
		}
		return storage.NewResourceExistsError("component", "unknown").
			WithDetail("operation", operation)

	case errors.As(err, &throttledErr):
		// Throttling error
		return storage.NewThrottledError(err.Error()).
			WithDetail("operation", operation)

	default:
		// Generic storage error
		return storage.NewStorageUnavailableError(err.Error()).
			WithDetail("operation", operation)
	}
}

// convertStorageConfig converts the generic storage config to DynamoDB-specific config
func convertStorageConfig(storageConfig *storage.DynamoDBStorageConfig) (*Config, error) {
	config := &Config{
		TableName:         storageConfig.TableName,
		Region:            storageConfig.Region,
		Endpoint:          storageConfig.Endpoint,
		ConsistentReads:   storageConfig.ConsistentReads,
		MaxRetries:        storageConfig.MaxRetries,
		MaxBatchSize:      storageConfig.MaxBatchSize,
		AutoCreateTable:   storageConfig.AutoCreateTable,
		VerifyTableSchema: storageConfig.VerifyTableSchema,
	}

	// Parse timeout duration
	if storageConfig.QueryTimeout != "" {
		timeout, err := time.ParseDuration(storageConfig.QueryTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid query timeout: %w", err)
		}
		config.QueryTimeout = timeout
	} else {
		config.QueryTimeout = 30 * time.Second
	}

	// Set defaults
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.MaxBatchSize == 0 {
		config.MaxBatchSize = 25
	}

	return config, nil
}

// Additional helper methods for filtering, sorting, and table management

func (s *componentStore) buildScanInput(filters *storage.ComponentFilters, pagination *storage.Pagination) *dynamodb.ScanInput {
	input := &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	}

	if pagination.Limit > 0 {
		input.Limit = aws.Int32(pagination.Limit)
	}

	// Handle pagination token
	if pagination.NextToken != "" {
		if lastKey, err := s.decodeToken(pagination.NextToken); err == nil && lastKey != nil {
			input.ExclusiveStartKey = lastKey
		}
	}

	// Build filter expressions (simplified for MVP)
	var filterParts []string
	var expressionValues map[string]types.AttributeValue

	if filters != nil {
		expressionValues = make(map[string]types.AttributeValue)

		// Provider filter
		if len(filters.Providers) > 0 {
			var providerParts []string
			for i, provider := range filters.Providers {
				key := fmt.Sprintf(":provider%d", i)
				providerParts = append(providerParts, fmt.Sprintf("Provider = %s", key))
				expressionValues[key] = &types.AttributeValueMemberS{Value: provider}
			}
			if len(providerParts) > 0 {
				filterParts = append(filterParts, fmt.Sprintf("(%s)", fmt.Sprintf("%v", providerParts)))
			}
		}

		// Category filter
		if len(filters.Categories) > 0 {
			var categoryParts []string
			for i, category := range filters.Categories {
				key := fmt.Sprintf(":category%d", i)
				categoryParts = append(categoryParts, fmt.Sprintf("Category = %s", key))
				expressionValues[key] = &types.AttributeValueMemberS{Value: category}
			}
			if len(categoryParts) > 0 {
				filterParts = append(filterParts, fmt.Sprintf("(%s)", fmt.Sprintf("%v", categoryParts)))
			}
		}
	}

	// Apply filter expression if we have filters
	if len(filterParts) > 0 {
		input.FilterExpression = aws.String(fmt.Sprintf("%v", filterParts))
		input.ExpressionAttributeValues = expressionValues
	}

	return input
}

func (s *componentStore) applyPostScanFilters(components []*models.ComponentDefinition, filters *storage.ComponentFilters) []*models.ComponentDefinition {
	if filters == nil {
		return components
	}

	var filtered []*models.ComponentDefinition
	for _, component := range components {
		if s.matchesFilters(component, filters) {
			filtered = append(filtered, component)
		}
	}
	return filtered
}

func (s *componentStore) matchesFilters(component *models.ComponentDefinition, filters *storage.ComponentFilters) bool {
	// Active only filter
	if filters.ActiveOnly && component.IsDeprecated() {
		return false
	}

	// Date filters
	if filters.CreatedAfter != nil && component.Metadata.CreatedAt.Before(*filters.CreatedAfter) {
		return false
	}
	if filters.CreatedBefore != nil && component.Metadata.CreatedAt.After(*filters.CreatedBefore) {
		return false
	}

	// Label filters
	if len(filters.Labels) > 0 {
		for key, value := range filters.Labels {
			if !component.HasLabel(key, value) {
				return false
			}
		}
	}

	return true
}

func (s *componentStore) applySorting(components []*models.ComponentDefinition, sortBy storage.SortField, sortOrder storage.SortOrder) []*models.ComponentDefinition {
	// Simple sorting implementation for MVP
	// In production, this should be done at the database level for better performance
	return components
}

func (s *componentStore) encodeToken(lastKey map[string]types.AttributeValue) string {
	// Convert DynamoDB LastEvaluatedKey to JSON and encode as base64
	if lastKey == nil {
		return ""
	}

	// Convert to a map that can be marshaled to JSON
	jsonMap := make(map[string]interface{})
	for k, v := range lastKey {
		switch tv := v.(type) {
		case *types.AttributeValueMemberS:
			jsonMap[k] = map[string]string{"S": tv.Value}
		case *types.AttributeValueMemberN:
			jsonMap[k] = map[string]string{"N": tv.Value}
		case *types.AttributeValueMemberB:
			jsonMap[k] = map[string][]byte{"B": tv.Value}
			// Add other types as needed
		}
	}

	jsonData, err := json.Marshal(jsonMap)
	if err != nil {
		s.logger.Error("failed to marshal pagination token", "error", err)
		return ""
	}

	return base64.StdEncoding.EncodeToString(jsonData)
}

func (s *componentStore) decodeToken(token string) (map[string]types.AttributeValue, error) {
	if token == "" {
		return nil, nil
	}

	// Decode base64 token
	jsonData, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination token: %w", err)
	}

	// Unmarshal JSON to map
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonMap); err != nil {
		return nil, fmt.Errorf("invalid pagination token format: %w", err)
	}

	// Convert back to DynamoDB AttributeValue map
	result := make(map[string]types.AttributeValue)
	for k, v := range jsonMap {
		valueMap, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		if strVal, ok := valueMap["S"].(string); ok {
			result[k] = &types.AttributeValueMemberS{Value: strVal}
		} else if numVal, ok := valueMap["N"].(string); ok {
			result[k] = &types.AttributeValueMemberN{Value: numVal}
		} else if binVal, ok := valueMap["B"].([]byte); ok {
			result[k] = &types.AttributeValueMemberB{Value: binVal}
		}
		// Add other types as needed
	}

	return result, nil
}

func (s *componentStore) ensureTable(ctx context.Context) error {
	exists, err := s.client.TableExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if exists {
		s.logger.InfoContext(ctx, "table already exists", "table", s.tableName)
		return nil
	}

	s.logger.InfoContext(ctx, "creating table", "table", s.tableName)

	// Create table with basic schema
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
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Wait for table to be active
	return s.client.WaitForTable(ctx)
}
