package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notification-service/internal/app"
)

func main() {
	amqpURL := getEnv("AMQP_URL", "amqp://guest:guest@localhost:5672/")

	// Retry app initialization (RabbitMQ / Redis may not be ready yet)
	var application *app.App
	var err error
	for i := 0; i < 10; i++ {
		application, err = app.NewApp(amqpURL)
		if err == nil {
			break
		}
		log.Printf("[Main] App init failed (%v), retrying in 3s…", err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("[Main] Could not initialize app: %v", err)
	}
	defer application.Consumer.Close()
	defer application.RedisClient.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Printf("Received signal %s – initiating graceful shutdown", sig)
		cancel()
	}()

	log.Println("Notification Service started (Assignment 4 – Redis Idempotency + Adapter Pattern + Exponential Backoff)")
	if err := application.Consumer.Run(ctx); err != nil {
		log.Printf("Consumer stopped with error: %v", err)
	}
	log.Println("Notification Service shut down gracefully")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func init() {
	fmt.Println() // ensure log prefix is visible
}
