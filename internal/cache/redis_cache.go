package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Get(ctx context.Context, key string, value interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Close() error
}

type redisCache struct {
	client *redis.Client
	cfg    *config.CacheConfig
}

func NewRedisCache(client *redis.Client, cfg *config.CacheConfig) Cache {
	return &redisCache{
		client: client,
		cfg:    cfg,
	}
}

func (r *redisCache) Get(ctx context.Context, key string, value any) (bool, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get key %s from redis: %w", key, err)
	}

	if err := json.Unmarshal(data, value); err != nil {
		return false, fmt.Errorf("failed to unmarshal cache data for key %s: %w", key, err)
	}

	return true, nil
}

func (r *redisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}

	if ttl <= 0 {
		ttl = r.cfg.DefaultTTL
	}

	err = r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set key %s in redis: %w", key, err)
	}

	return nil
}

func (r *redisCache) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key %s from redis: %w", key, err)
	}

	return nil
}

func (r *redisCache) Close() error {
	return nil
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
