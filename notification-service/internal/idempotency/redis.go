package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultTTL = 24 * time.Hour

// Status values stored in Redis.
const (
	StatusProcessed = "processed"
	StatusFailed    = "failed"
)

// RedisIdempotencyStore checks and records whether a payment_id has been processed.
// This prevents sending duplicate notifications when a job is retried.
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

// IsProcessed returns true if this payment_id was already successfully handled.
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

// MarkProcessed stores the payment_id with a "processed" status so future retries are skipped.
func (s *RedisIdempotencyStore) MarkProcessed(ctx context.Context, paymentID string) error {
	return s.client.Set(ctx, s.key(paymentID), StatusProcessed, s.ttl).Err()
}
