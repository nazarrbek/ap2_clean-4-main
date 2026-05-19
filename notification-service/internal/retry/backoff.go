package retry

import (
	"context"
	"log"
	"time"
)

type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func DefaultConfig() Config {
	return Config{
		MaxAttempts: 5,
		BaseDelay:   2 * time.Second,
		MaxDelay:    30 * time.Second,
	}
}

func Do(ctx context.Context, cfg Config, name string, fn func() error) error {
	var lastErr error
	delay := cfg.BaseDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			if attempt > 1 {
				log.Printf("[Retry] %s succeeded on attempt %d", name, attempt)
			}
			return nil
		}

		log.Printf("[Retry] %s failed (attempt %d/%d): %v", name, attempt, cfg.MaxAttempts, lastErr)

		if attempt == cfg.MaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			log.Printf("[Retry] %s: context cancelled, stopping retries", name)
			return ctx.Err()
		case <-time.After(delay):
		}

		delay *= 2
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	log.Printf("[Retry] %s exhausted all %d attempts, last error: %v", name, cfg.MaxAttempts, lastErr)
	return lastErr
}
