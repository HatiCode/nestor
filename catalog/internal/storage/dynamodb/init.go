package dynamodb

import (
	"github.com/HatiCode/nestor/catalog/internal/storage"
	"github.com/HatiCode/nestor/catalog/pkg/cache"
	"github.com/HatiCode/nestor/shared/pkg/logging"
)

// RegisterWith registers the DynamoDB component store factory with the provided registry.
func RegisterWith(registry *storage.Registry) {
	registry.Register("dynamodb", func(config *storage.StorageConfig, cache cache.Cache, logger logging.Logger) (storage.ComponentStore, error) {
		return NewComponentStore(config, cache, logger)
	})
}

func init() {
	// Register with the default registry for backward compatibility
	RegisterWith(storage.DefaultRegistry)
}
