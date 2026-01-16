package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	Client *redis.Client
}

func NewCache(client *redis.Client) *Cache {
	return &Cache{
		Client: client,
	}
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	err := c.Client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set in redis: %w", err)
	}
	return nil
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get from redis: %w", err)
	}
	return val, nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	err := c.Client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete from redis: %w", err)
	}
	return nil
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence in redis: %w", err)
	}
	return result > 0, nil
}
