package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"payment-service/internal/app"
)

func main() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "payments_db")
	httpPort := getEnv("SERVER_PORT", "8081")
	grpcPort := getEnv("GRPC_PORT", "50051")
	amqpURL := getEnv("AMQP_URL", "amqp://guest:guest@localhost:5672/")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	db, err := openDB(dsn)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	application, err := app.NewApp(db, amqpURL)
	if err != nil {
		log.Fatalf("app init: %v", err)
	}
	defer application.Publisher.Close()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	grpcLis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen for grpc: %v", err)
	}

	go func() {
		log.Printf("Payment Service gRPC server starting on :%s", grpcPort)
		if err := application.GRPCServer.Serve(grpcLis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	go func() {
		sig := <-sigs
		log.Printf("Received signal %s – graceful shutdown", sig)
		application.GRPCServer.GracefulStop()
		application.Publisher.Close()
		os.Exit(0)
	}()

	log.Printf("Payment Service HTTP server starting on :%s", httpPort)
	if err := application.Router.Run(":" + httpPort); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	for i := 0; i < 10; i++ {
		db, err := sql.Open("postgres", dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				log.Println("Connected to payments database")
				return db, nil
			}
		}
		log.Printf("DB not ready, retrying in 3s…")
		time.Sleep(3 * time.Second)
	}
	return nil, fmt.Errorf("could not connect to database after retries")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
