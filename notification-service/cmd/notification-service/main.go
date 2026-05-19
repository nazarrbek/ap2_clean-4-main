package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notification-service/internal/app"
)

func main() {
	amqpURL := getEnv("AMQP_URL", "amqp://guest:guest@localhost:5672/")

	var application *app.App
	var err error
	for i := 0; i < 10; i++ {
		application, err = app.NewApp(amqpURL)
		if err == nil {
			break
		}
		log.Printf("[Main] app init failed (%v), retrying in 3s", err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("[Main] could not initialize app: %v", err)
	}
	defer application.Consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Printf("[Main] received signal %s, shutting down", sig)
		cancel()
	}()

	log.Println("Notification Service started")
	if err := application.Consumer.Run(ctx); err != nil {
		log.Printf("[Main] consumer stopped with error: %v", err)
	}
	log.Println("Notification Service stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
