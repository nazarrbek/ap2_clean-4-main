package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"

	"notification-service/internal/adapter"
	"notification-service/internal/messaging"
	"notification-service/internal/repository"
	"notification-service/internal/usecase"
)

func main() {
	// ── Redis ────────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
	})
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	log.Println("✅ Redis connected")

	// ── RabbitMQ ─────────────────────────────────────────────────────────────
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	conn, err := dialWithRetry(rabbitURL, 10)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer conn.Close()
	log.Println("✅ RabbitMQ connected")

	// ── Adapter selection via env ─────────────────────────────────────────────
	// Set PROVIDER_MODE=REAL to use SMTP, default is SIMULATED.
	var emailSender adapter.EmailSender
	switch getEnv("PROVIDER_MODE", "SIMULATED") {
	case "REAL":
		emailSender = adapter.NewSMTPSender(
			getEnv("SMTP_HOST", "smtp.example.com"),
			getEnv("SMTP_PORT", "587"),
			getEnv("SMTP_USER", ""),
			getEnv("SMTP_PASS", ""),
		)
		log.Println("📧 Using REAL SMTP provider")
	default:
		failRate := getEnvFloat("SIMULATE_FAIL_RATE", 0.3)
		latencyMs := getEnvInt("SIMULATE_LATENCY_MS", 200)
		emailSender = adapter.NewSimulatedSender(failRate, time.Duration(latencyMs)*time.Millisecond)
		log.Printf("🎭 Using SIMULATED provider (fail_rate=%.0f%%, latency=%dms)", failRate*100, latencyMs)
	}

	// ── Wiring ────────────────────────────────────────────────────────────────
	idempotencyHours := getEnvInt("IDEMPOTENCY_TTL_HOURS", 24)
	idempotencyRepo := repository.NewIdempotencyRepository(rdb, time.Duration(idempotencyHours)*time.Hour)

	maxRetries := getEnvInt("MAX_RETRIES", 3)
	notifUC := usecase.NewNotificationUsecase(idempotencyRepo, emailSender, maxRetries)

	consumer := messaging.NewConsumer(conn, notifUC)

	// ── Run ───────────────────────────────────────────────────────────────────
	bgCtx, bgCancel := context.WithCancel(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := consumer.Start(bgCtx); err != nil {
			log.Printf("consumer error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down notification-service...")
	bgCancel()
	time.Sleep(1 * time.Second)
}

// dialWithRetry retries RabbitMQ connection with exponential backoff.
func dialWithRetry(url string, maxAttempts int) (*amqp.Connection, error) {
	for i := 1; i <= maxAttempts; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			return conn, nil
		}
		wait := time.Duration(1<<uint(i)) * time.Second
		log.Printf("rabbitmq not ready (attempt %d/%d): %v — retrying in %s", i, maxAttempts, err, wait)
		time.Sleep(wait)
	}
	return nil, fmt.Errorf("could not connect to rabbitmq after %d attempts", maxAttempts)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
