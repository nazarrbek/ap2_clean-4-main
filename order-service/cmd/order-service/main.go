package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"

	"order-service/internal/app"
)

func main() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "orders_db")
	paymentGRPCAddr := getEnv("PAYMENT_GRPC_ADDR", "localhost:50051")
	httpPort := getEnv("SERVER_PORT", "8080")
	grpcPort := getEnv("GRPC_PORT", "50052")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("Connected to orders database")

	application, err := app.NewApp(db, paymentGRPCAddr)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	grpcLis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen for grpc: %v", err)
	}

	go func() {
		log.Printf("Order Service gRPC server starting on :%s", grpcPort)
		if err := application.GRPCServer.Serve(grpcLis); err != nil {
			log.Fatalf("failed to start grpc server: %v", err)
		}
	}()

	log.Printf("Order Service HTTP server starting on :%s", httpPort)
	if err := application.Router.Run(":" + httpPort); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
