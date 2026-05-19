package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter holds configuration for the sliding-window rate limiter.
type RateLimiter struct {
	client   *redis.Client
	limit    int           // max requests
	window   time.Duration // time window
}

// NewRateLimiter creates a rate limiter middleware.
// Example: NewRateLimiter(client, 10, time.Minute) → 10 req/min per IP.
func NewRateLimiter(client *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{client: client, limit: limit, window: window}
}

// Middleware returns a gin.HandlerFunc that enforces the rate limit.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("ratelimit:%s", ip)
		ctx := context.Background()

		// Atomic increment + set TTL on first request
		pipe := rl.client.Pipeline()
		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, rl.window)
		_, err := pipe.Exec(ctx)

		if err != nil {
			// Redis error: fail open (don't block requests)
			c.Next()
			return
		}

		count := incr.Val()
		if count > int64(rl.limit) {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"limit":       rl.limit,
				"window":      rl.window.String(),
				"retry_after": rl.window.String(),
			})
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", int64(rl.limit)-count))
		c.Next()
	}
}
