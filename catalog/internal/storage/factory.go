package storage

import (
	"fmt"
	"strings"
	"time"

	"github.com/HatiCode/nestor/catalog/pkg/cache"
	"github.com/HatiCode/nestor/shared/pkg/logging"
)

// StorageConfig defines the configuration for storage backends.
type StorageConfig struct {
	Type     string                 `yaml:"type" validate:"required,oneof=dynamodb memory postgres"`
	DynamoDB *DynamoDBStorageConfig `yaml:"dynamodb,omitempty"`
}

// DynamoDBStorageConfig contains DynamoDB-specific configuration.
type DynamoDBStorageConfig struct {
	TableName         string `yaml:"table_name" json:"table_name"`
	Region            string `yaml:"region" json:"region" validate:"required"`
	Endpoint          string `yaml:"endpoint" json:"endpoint"`
	ConsistentReads   bool   `yaml:"consistent_reads" json:"consistent_reads"`
	QueryTimeout      string `yaml:"query_timeout" json:"query_timeout"`
	MaxRetries        int    `yaml:"max_retries" json:"max_retries"`
	MaxBatchSize      int    `yaml:"max_batch_size" json:"max_batch_size"`
	AutoCreateTable   bool   `yaml:"auto_create_table" json:"auto_create_table"`
	VerifyTableSchema bool   `yaml:"verify_table_schema" json:"verify_table_schema"`
}

// Validates the storage configuration.
func (c *StorageConfig) Validate() error {
	if c == nil {
		return NewConfigurationError("storage", "configuration is nil")
	}

	if c.Type == "" {
		return NewConfigurationError("type", "storage type is required")
	}

	switch c.Type {
	case "dynamodb":
		if c.DynamoDB == nil {
			return NewConfigurationError("dynamodb", "DynamoDB config is required when type is dynamodb")
		}
		return c.DynamoDB.Validate()
	case "memory":
		return nil
	case "postgres":
		return NewConfigurationError("type", "PostgreSQL implementation not yet available")
	default:
		return NewConfigurationError("type", fmt.Sprintf("unsupported storage type: %s", c.Type))
	}
}

// Validates the DynamoDB configuration.
func (c *DynamoDBStorageConfig) Validate() error {
	if c == nil {
		return NewConfigurationError("dynamodb", "DynamoDB config cannot be nil")
	}

	if c.Region == "" {
		return NewConfigurationError("region", "region is required for DynamoDB")
	}

	if c.MaxRetries < 0 {
		return NewConfigurationError("max_retries", "max_retries cannot be negative")
	}

	if c.MaxBatchSize < 1 || c.MaxBatchSize > 25 {
		return NewConfigurationError("max_batch_size", "max_batch_size must be between 1 and 25")
	}

	if c.QueryTimeout != "" {
		_, err := time.ParseDuration(c.QueryTimeout)
		if err != nil {
			return NewConfigurationError("query_timeout", fmt.Sprintf("invalid duration format: %v", err))
		}
	}

	if c.TableName != "" && len(c.TableName) < 3 {
		return NewConfigurationError("table_name", "table name must be at least 3 characters long")
	}

	if c.Endpoint != "" && !strings.HasPrefix(c.Endpoint, "http") {
		return NewConfigurationError("endpoint", "endpoint must be a valid URL starting with http:// or https://")
	}

	return nil
}

// ComponentStoreFactory is a function type that creates a ComponentStore.
type ComponentStoreFactory func(config *StorageConfig, cache cache.Cache, logger logging.Logger) (ComponentStore, error)

// Registry holds the registered component store factories.
type Registry struct {
	factories map[string]ComponentStoreFactory
}

// NewRegistry creates a new registry with optional pre-registered factories.
func NewRegistry(factories map[string]ComponentStoreFactory) *Registry {
	if factories == nil {
		factories = make(map[string]ComponentStoreFactory)
	}
	return &Registry{
		factories: factories,
	}
}

// Register registers a factory for a specific storage type.
func (r *Registry) Register(storageType string, factory ComponentStoreFactory) {
	r.factories[storageType] = factory
}

// Create creates a new ComponentStore based on configuration.
func (r *Registry) Create(config *StorageConfig, cache cache.Cache, logger logging.Logger) (ComponentStore, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid storage configuration: %w", err)
	}

	factory, exists := r.factories[config.Type]
	if !exists {
		return nil, NewConfigurationError("type", fmt.Sprintf("no factory registered for storage type: %s", config.Type))
	}

	store, err := factory(config, cache, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s component store: %w", config.Type, err)
	}

	return store, nil
}

// DefaultRegistry is a convenience instance for backward compatibility.
var DefaultRegistry = NewRegistry(nil)

// RegisterComponentStoreFactory registers a factory with the default registry.
// This is provided for backward compatibility
func RegisterComponentStoreFactory(storageType string, factory ComponentStoreFactory) {
	DefaultRegistry.Register(storageType, factory)
}

// NewComponentStore creates a new ComponentStore using the default registry.
// This is provided for backward compatibility
func NewComponentStore(config *StorageConfig, cache cache.Cache, logger logging.Logger) (ComponentStore, error) {
	return DefaultRegistry.Create(config, cache, logger)
}
