package cache

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string, value interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Close() error
}

func Key(prefix string, id string) string {
	return prefix + ":" + id
}

const (
	ProductKeyPrefix = "product"
	UserKeyPrefix    = "user"
	OrderKeyPrefix   = "order"
	CartKeyPrefix    = "cart"
)
