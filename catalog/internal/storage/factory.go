package storage

import (
	"github.com/HatiCode/nestor/catalog/internal/storage/dynamodb"
	"github.com/HatiCode/nestor/shared/pkg/logging"
)

type StorageConfig struct {
	Type     string           `yaml:"type" validate:"required,oneof=dynamodb memory postgres"`
	DynamoDB *dynamodb.Config `yaml:"dynamodb,omitempty"`
}

func NewCatalogStore(config *StorageConfig, cache Cache, logger logging.Logger) (CatalogStore, error) {
	switch config.Type {
	case "dynamodb":
		if config.DynamoDB == nil {
			return nil, ErrInvalidConfig.WithDetail("reason", "DynamoDB config is required")
		}
		return dynamodb.NewCatalogStore(config.DynamoDB, cache, logger)
	case "memory":
		return NewMemoryStore(cache, logger)
	case "postgres":
		return nil, ErrUnsupportedStorageType.WithDetail("type", "postgres").
			WithDetail("reason", "PostgreSQL implementation not yet available")
	default:
		return nil, ErrUnsupportedStorageType.WithDetail("type", config.Type)
	}
}

// NewMemoryStore creates a new in-memory catalog store for testing
func NewMemoryStore(cache Cache, logger logging.Logger) (CatalogStore, error) {
	// Implementation will be in memory/catalog.go
	panic("not implemented - to be implemented in memory package")
}

var (
	ErrInvalidConfig = NewStorageError("INVALID_CONFIG", "invalid storage configuration")
)
