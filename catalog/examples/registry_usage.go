package examples

import (
	"github.com/HatiCode/nestor/catalog/internal/storage"
	"github.com/HatiCode/nestor/catalog/internal/storage/dynamodb"
	"github.com/HatiCode/nestor/catalog/pkg/cache"
	"github.com/HatiCode/nestor/shared/pkg/logging"
)

// This example demonstrates how to use the registry pattern for dependency injection

func ExampleRegistryUsage() {
	// Create a new registry
	registry := storage.NewRegistry(nil)

	// Register storage implementations
	dynamodb.RegisterWith(registry)

	// In a real application, you might register other implementations:
	// memory.RegisterWith(registry)
	// postgres.RegisterWith(registry)

	// Create a logger
	logger := logging.NewLogger()

	// Create a cache
	cacheClient := cache.NewMemoryCache()

	// Create a storage config
	config := &storage.StorageConfig{
		Type: "dynamodb",
		DynamoDB: &storage.DynamoDBStorageConfig{
			Region:    "us-west-2",
			TableName: "components",
		},
	}

	// Create a component store using the registry
	componentStore, err := registry.Create(config, cacheClient, logger)
	if err != nil {
		logger.Error("Failed to create component store", "error", err)
		return
	}

	// Use the component store
	_ = componentStore
}

// Example of using the registry in a service constructor with dependency injection
type CatalogService struct {
	store  storage.ComponentStore
	logger logging.Logger
}

func NewCatalogService(store storage.ComponentStore, logger logging.Logger) *CatalogService {
	return &CatalogService{
		store:  store,
		logger: logger,
	}
}

func ExampleServiceWithDI() {
	// Create dependencies
	logger := logging.NewLogger()
	cacheClient := cache.NewMemoryCache()

	// Create registry and register implementations
	registry := storage.NewRegistry(nil)
	dynamodb.RegisterWith(registry)

	// Create config
	config := &storage.StorageConfig{
		Type: "dynamodb",
		DynamoDB: &storage.DynamoDBStorageConfig{
			Region:    "us-west-2",
			TableName: "components",
		},
	}

	// Create store
	store, err := registry.Create(config, cacheClient, logger)
	if err != nil {
		logger.Error("Failed to create component store", "error", err)
		return
	}

	// Create service with dependencies injected
	service := NewCatalogService(store, logger)

	// Use the service
	_ = service
}
