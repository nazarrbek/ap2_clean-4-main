package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"order-service/internal/domain"
)

// OrderCache defines caching operations — use cases depend only on this interface.
type OrderCache interface {
	Get(ctx context.Context, id uint) (*domain.Order, error)
	Set(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id uint) error
}

type redisOrderCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewOrderCache(client *redis.Client, ttl time.Duration) OrderCache {
	return &redisOrderCache{client: client, ttl: ttl}
}

func (c *redisOrderCache) key(id uint) string {
	return fmt.Sprintf("order:%d", id)
}

// Get implements the cache-aside READ path.
// Returns nil, nil when there is a cache miss (key not found).
func (c *redisOrderCache) Get(ctx context.Context, id uint) (*domain.Order, error) {
	data, err := c.client.Get(ctx, c.key(id)).Bytes()
	if err == redis.Nil {
		return nil, nil // cache miss — caller should fall back to DB
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}
	var order domain.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}
	return &order, nil
}

// Set stores an order in Redis with TTL.
func (c *redisOrderCache) Set(ctx context.Context, order *domain.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal order: %w", err)
	}
	return c.client.Set(ctx, c.key(order.ID), data, c.ttl).Err()
}

// Delete removes the key — called on status changes (invalidation).
func (c *redisOrderCache) Delete(ctx context.Context, id uint) error {
	return c.client.Del(ctx, c.key(id)).Err()
}
