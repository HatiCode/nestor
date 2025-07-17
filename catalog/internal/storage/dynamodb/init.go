package dynamodb

import (
	"github.com/HatiCode/nestor/catalog/internal/storage"
	"github.com/HatiCode/nestor/catalog/pkg/cache"
	"github.com/HatiCode/nestor/shared/pkg/logging"
)

func init() {
	// Register the DynamoDB component store factory
	storage.RegisterComponentStoreFactory("dynamodb", func(config *storage.StorageConfig, cache cache.Cache, logger logging.Logger) (storage.ComponentStore, error) {
		return NewComponentStore(config, cache, logger)
	})
}
