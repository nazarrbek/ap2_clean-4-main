package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"order-service/internal/domain"
)

const defaultTTL = 5 * time.Minute

// OrderCache is the interface the use-case layer depends on.
// It hides all Redis details from business logic.
type OrderCache interface {
	Get(ctx context.Context, id string) (*domain.Order, error)
	Set(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id string) error
}

// RedisOrderCache implements OrderCache using go-redis.
type RedisOrderCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisOrderCache(client *redis.Client, ttl time.Duration) *RedisOrderCache {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	return &RedisOrderCache{client: client, ttl: ttl}
}

func (c *RedisOrderCache) key(id string) string {
	return fmt.Sprintf("order:%s", id)
}

// Get retrieves an order from Redis. Returns (nil, nil) on cache miss.
func (c *RedisOrderCache) Get(ctx context.Context, id string) (*domain.Order, error) {
	val, err := c.client.Get(ctx, c.key(id)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil // cache miss – not an error
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var order domain.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}
	return &order, nil
}

// Set stores an order in Redis with the configured TTL.
func (c *RedisOrderCache) Set(ctx context.Context, order *domain.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal order: %w", err)
	}
	if err := c.client.Set(ctx, c.key(order.ID), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

// Delete removes an order from Redis (cache invalidation).
func (c *RedisOrderCache) Delete(ctx context.Context, id string) error {
	if err := c.client.Del(ctx, c.key(id)).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}
