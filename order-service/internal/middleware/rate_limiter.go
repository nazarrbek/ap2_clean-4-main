package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a sliding-window counter per client IP using Redis INCR + EXPIRE.
// This demonstrates Redis as a shared state store for distributed systems.
type RateLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
}

type Option func(*RateLimiter)

func WithLimit(n int) Option        { return func(r *RateLimiter) { r.limit = n } }
func WithWindow(d time.Duration) Option { return func(r *RateLimiter) { r.window = d } }

func NewRateLimiter(client *redis.Client, opts ...Option) *RateLimiter {
	rl := &RateLimiter{client: client, limit: 10, window: time.Minute}
	for _, o := range opts {
		o(rl)
	}
	return rl
}

// Middleware returns an http.Handler middleware that enforces rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = forwarded
		}
		key := fmt.Sprintf("rate:%s", clientIP)

		ctx := context.Background()
		count, err := rl.client.Incr(ctx, key).Result()
		if err != nil {
			log.Printf("rate limiter redis error: %v", err)
			next.ServeHTTP(w, r) // fail open
			return
		}
		// Set expiry only on first request
		if count == 1 {
			rl.client.Expire(ctx, key, rl.window)
		}
		// Add headers so the client can see their quota
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, rl.limit-int(count))))

		if int(count) > rl.limit {
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
