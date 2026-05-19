package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultTTL = 24 * time.Hour

const (
	StatusProcessed = "processed"
	StatusFailed    = "failed"
)

type RedisIdempotencyStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisIdempotencyStore(client *redis.Client, ttl time.Duration) *RedisIdempotencyStore {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	return &RedisIdempotencyStore{client: client, ttl: ttl}
}

func (s *RedisIdempotencyStore) key(paymentID string) string {
	return fmt.Sprintf("notification:idempotency:%s", paymentID)
}

func (s *RedisIdempotencyStore) IsProcessed(ctx context.Context, paymentID string) (bool, error) {
	val, err := s.client.Get(ctx, s.key(paymentID)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis get idempotency: %w", err)
	}
	return val == StatusProcessed, nil
}

func (s *RedisIdempotencyStore) MarkProcessed(ctx context.Context, paymentID string) error {
	return s.client.Set(ctx, s.key(paymentID), StatusProcessed, s.ttl).Err()
}
