package app

import (
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"notification-service/internal/idempotency"
	"notification-service/internal/messaging"
	"notification-service/internal/provider"
	"notification-service/internal/retry"
	"notification-service/internal/usecase"
)

type App struct {
	Consumer    *messaging.Consumer
	RedisClient *redis.Client
}

func NewApp(amqpURL string) (*App, error) {
	// ── Redis ────────────────────────────────────────────────────────────
	redisClient := newRedisClient()

	idempotencyTTL := parseDuration(os.Getenv("IDEMPOTENCY_TTL"), 24*time.Hour)
	idempotencyStore := idempotency.NewRedisIdempotencyStore(redisClient, idempotencyTTL)

	// ── Provider (Adapter Pattern) ────────────────────────────────────────
	// Choose provider implementation via PROVIDER_MODE env var (REAL / SIMULATED)
	emailSender := newEmailProvider()

	// ── Retry config ─────────────────────────────────────────────────────
	retryCfg := retry.Config{
		MaxAttempts: parseInt(os.Getenv("RETRY_MAX_ATTEMPTS"), 5),
		BaseDelay:   parseDuration(os.Getenv("RETRY_BASE_DELAY"), 2*time.Second),
		MaxDelay:    parseDuration(os.Getenv("RETRY_MAX_DELAY"), 30*time.Second),
	}

	// ── Use Case ─────────────────────────────────────────────────────────
	uc := usecase.NewNotificationUseCase(emailSender, idempotencyStore, retryCfg)

	// ── Consumer ─────────────────────────────────────────────────────────
	consumer, err := messaging.NewConsumer(amqpURL, uc)
	if err != nil {
		return nil, err
	}

	return &App{Consumer: consumer, RedisClient: redisClient}, nil
}

// newEmailProvider selects the provider adapter based on PROVIDER_MODE env var.
func newEmailProvider() provider.EmailSender {
	mode := os.Getenv("PROVIDER_MODE")
	switch mode {
	case "REAL":
		p := provider.NewSMTPProvider()
		fmt.Println("[App] Using REAL SMTP email provider")
		return p
	default:
		p := provider.NewSimulatedProvider()
		fmt.Println("[App] Using SIMULATED email provider (set PROVIDER_MODE=REAL for real email)")
		return p
	}
}

func newRedisClient() *redis.Client {
	host := getEnv("REDIS_HOST", "localhost")
	port := getEnv("REDIS_PORT", "6379")
	password := os.Getenv("REDIS_PASSWORD")

	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       0,
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func parseInt(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return fallback
	}
	return n
}
