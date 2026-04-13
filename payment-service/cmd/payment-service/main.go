package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

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
	log.Println("Connected to payments database")

	application := app.NewApp(db)

	grpcLis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen for grpc: %v", err)
	}

	go func() {
		log.Printf("Payment Service gRPC server starting on :%s", grpcPort)
		if err := application.GRPCServer.Serve(grpcLis); err != nil {
			log.Fatalf("failed to start grpc server: %v", err)
		}
	}()

	log.Printf("Payment Service HTTP server starting on :%s", httpPort)
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
