package app

import (
	"fmt"
	"os"
	"time"

	"notification-service/internal/idempotency"
	"notification-service/internal/messaging"
	"notification-service/internal/provider"
	"notification-service/internal/retry"
	"notification-service/internal/usecase"
)

type App struct {
	Consumer *messaging.Consumer
}

func NewApp(amqpURL string) (*App, error) {
	idempotencyTTL := parseDuration(os.Getenv("IDEMPOTENCY_TTL"), 24*time.Hour)
	idempotencyStore := idempotency.NewMemoryIdempotencyStore(idempotencyTTL)

	emailSender := newEmailProvider()

	retryCfg := retry.Config{
		MaxAttempts: parseInt(os.Getenv("RETRY_MAX_ATTEMPTS"), 5),
		BaseDelay:   parseDuration(os.Getenv("RETRY_BASE_DELAY"), 2*time.Second),
		MaxDelay:    parseDuration(os.Getenv("RETRY_MAX_DELAY"), 30*time.Second),
	}

	uc := usecase.NewNotificationUseCase(emailSender, idempotencyStore, retryCfg)

	consumer, err := messaging.NewConsumer(amqpURL, uc)
	if err != nil {
		return nil, err
	}

	return &App{Consumer: consumer}, nil
}

func newEmailProvider() provider.EmailSender {
	mode := os.Getenv("PROVIDER_MODE")
	switch mode {
	case "REAL":
		p := provider.NewSMTPProvider()
		fmt.Println("[App] using real SMTP provider")
		return p
	default:
		p := provider.NewSimulatedProvider()
		fmt.Println("[App] using simulated email provider")
		return p
	}
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
