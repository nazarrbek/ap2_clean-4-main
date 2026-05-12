package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// IdempotencyStatus values stored in Redis.
const (
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusFailed     = "failed"
)

// IdempotencyRepository manages deduplication records in Redis.
// Using Redis for idempotency allows the check to be shared across
// multiple worker replicas without a single point of contention.
type IdempotencyRepository interface {
	IsProcessed(ctx context.Context, paymentID uint) (bool, error)
	MarkProcessing(ctx context.Context, paymentID uint) error
	MarkDone(ctx context.Context, paymentID uint) error
	MarkFailed(ctx context.Context, paymentID uint) error
}

type redisIdempotencyRepo struct {
	client *redis.Client
	ttl    time.Duration
}

func NewIdempotencyRepository(client *redis.Client, ttl time.Duration) IdempotencyRepository {
	return &redisIdempotencyRepo{client: client, ttl: ttl}
}

func (r *redisIdempotencyRepo) key(paymentID uint) string {
	return fmt.Sprintf("notif:idempotency:%d", paymentID)
}

// IsProcessed returns true if the payment was already successfully processed.
func (r *redisIdempotencyRepo) IsProcessed(ctx context.Context, paymentID uint) (bool, error) {
	val, err := r.client.Get(ctx, r.key(paymentID)).Result()
	if err == redis.Nil {
		return false, nil // not processed yet
	}
	if err != nil {
		return false, fmt.Errorf("redis get: %w", err)
	}
	return val == StatusDone, nil
}

func (r *redisIdempotencyRepo) MarkProcessing(ctx context.Context, paymentID uint) error {
	// NX: only set if not exists — prevents concurrent duplicates
	return r.client.SetNX(ctx, r.key(paymentID), StatusProcessing, r.ttl).Err()
}

func (r *redisIdempotencyRepo) MarkDone(ctx context.Context, paymentID uint) error {
	return r.client.Set(ctx, r.key(paymentID), StatusDone, r.ttl).Err()
}

func (r *redisIdempotencyRepo) MarkFailed(ctx context.Context, paymentID uint) error {
	// Remove the key so the job can be retried on next delivery
	return r.client.Del(ctx, r.key(paymentID)).Err()
}
