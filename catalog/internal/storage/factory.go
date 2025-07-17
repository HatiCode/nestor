package storage

import (
	"fmt"
	"strings"
	"time"

	"github.com/HatiCode/nestor/catalog/pkg/cache"
	"github.com/HatiCode/nestor/shared/pkg/logging"
)

type StorageConfig struct {
	Type     string                 `yaml:"type" validate:"required,oneof=dynamodb memory postgres"`
	DynamoDB *DynamoDBStorageConfig `yaml:"dynamodb,omitempty"`
}

// DynamoDBStorageConfig contains DynamoDB-specific configuration
// This avoids the import cycle by not importing the dynamodb package directly
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

func (c *StorageConfig) Validate() error {
	if c == nil {
		return ErrInvalidConfig.WithDetail("reason", "storage configuration is nil")
	}

	if c.Type == "" {
		return ErrInvalidConfig.WithDetail("field", "type").WithDetail("reason", "storage type is required")
	}

	switch c.Type {
	case "dynamodb":
		if c.DynamoDB == nil {
			return ErrInvalidConfig.WithDetail("field", "dynamodb").WithDetail("reason", "DynamoDB config is required when type is dynamodb")
		}
		return c.DynamoDB.Validate()
	case "memory":
		// No additional validation needed for memory store
		return nil
	case "postgres":
		return ErrUnsupportedStorageType.WithDetail("type", "postgres").
			WithDetail("reason", "PostgreSQL implementation not yet available")
	default:
		return ErrUnsupportedStorageType.WithDetail("type", c.Type)
	}
}

func (c *DynamoDBStorageConfig) Validate() error {
	if c == nil {
		return ErrInvalidConfig.WithDetail("reason", "DynamoDB config cannot be nil")
	}

	if c.Region == "" {
		return ErrInvalidConfig.WithDetail("field", "region").WithDetail("reason", "region is required for DynamoDB")
	}

	if c.MaxRetries < 0 {
		return ErrInvalidConfig.WithDetail("field", "max_retries").WithDetail("reason", "max_retries cannot be negative")
	}

	if c.MaxBatchSize < 1 || c.MaxBatchSize > 25 {
		return ErrInvalidConfig.WithDetail("field", "max_batch_size").WithDetail("reason", "max_batch_size must be between 1 and 25")
	}

	// Validate query timeout if provided
	if c.QueryTimeout != "" {
		_, err := time.ParseDuration(c.QueryTimeout)
		if err != nil {
			return ErrInvalidConfig.WithDetail("field", "query_timeout").
				WithDetail("reason", fmt.Sprintf("invalid duration format: %v", err))
		}
	}

	// Validate table name if provided
	if c.TableName != "" && len(c.TableName) < 3 {
		return ErrInvalidConfig.WithDetail("field", "table_name").
			WithDetail("reason", "table name must be at least 3 characters long")
	}

	// Validate endpoint URL if provided
	if c.Endpoint != "" && !strings.HasPrefix(c.Endpoint, "http") {
		return ErrInvalidConfig.WithDetail("field", "endpoint").
			WithDetail("reason", "endpoint must be a valid URL starting with http:// or https://")
	}

	return nil
}

// Factory function that will be implemented by specific storage backends
type ComponentStoreFactory func(config *StorageConfig, cache cache.Cache, logger logging.Logger) (ComponentStore, error)

var componentStoreFactories = make(map[string]ComponentStoreFactory)

// RegisterComponentStoreFactory registers a factory for a specific storage type
func RegisterComponentStoreFactory(storageType string, factory ComponentStoreFactory) {
	componentStoreFactories[storageType] = factory
}

// NewComponentStore creates a new ComponentStore based on configuration
func NewComponentStore(config *StorageConfig, cache cache.Cache, logger logging.Logger) (ComponentStore, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid storage configuration: %w", err)
	}

	factory, exists := componentStoreFactories[config.Type]
	if !exists {
		return nil, ErrUnsupportedStorageType.WithDetail("type", config.Type).
			WithDetail("reason", fmt.Sprintf("no factory registered for storage type: %s", config.Type))
	}

	store, err := factory(config, cache, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s component store: %w", config.Type, err)
	}

	return store, nil
}
