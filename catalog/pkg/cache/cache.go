package cache

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) any
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) bool
}
