package app

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	orderpb "order-contract/order"

	"order-service/internal/cache"
	"order-service/internal/middleware"
	"order-service/internal/repository"
	ordergrpc "order-service/internal/transport/grpc"
	orderhttp "order-service/internal/transport/http"
	"order-service/internal/usecase"
)

type App struct {
	Router      *gin.Engine
	GRPCServer  *grpc.Server
	DB          *sql.DB
	RedisClient *redis.Client
}

func NewApp(db *sql.DB, paymentGRPCAddr string) (*App, error) {
	// ── Redis ────────────────────────────────────────────────────────────
	redisClient, err := newRedisClient()
	if err != nil {
		return nil, fmt.Errorf("redis: %w", err)
	}

	cacheTTL := parseDuration(os.Getenv("CACHE_TTL"), 5*time.Minute)
	orderCache := cache.NewRedisOrderCache(redisClient, cacheTTL)

	// ── Repository & Use Case ────────────────────────────────────────────
	orderRepo := repository.NewPostgresOrderRepository(db)

	paymentClient, err := ordergrpc.NewPaymentGRPCClient(paymentGRPCAddr, 2*time.Second)
	if err != nil {
		return nil, err
	}

	orderUC := usecase.NewOrderUseCase(orderRepo, orderCache, paymentClient)

	// ── HTTP ─────────────────────────────────────────────────────────────
	handler := orderhttp.NewOrderHandler(orderUC)
	router := gin.Default()

	// Bonus: Rate limiter middleware (10 req/min per IP by default)
	rlLimit := parseInt(os.Getenv("RATE_LIMIT_REQUESTS"), 10)
	rlWindow := parseDuration(os.Getenv("RATE_LIMIT_WINDOW"), time.Minute)
	rateLimiter := middleware.NewRateLimiter(redisClient, rlLimit, rlWindow)
	router.Use(rateLimiter.Middleware())

	handler.RegisterRoutes(router)

	// ── gRPC ─────────────────────────────────────────────────────────────
	orderStreamServer := ordergrpc.NewOrderGRPCServer(orderUC)
	grpcServer := grpc.NewServer()
	orderpb.RegisterOrderServiceServer(grpcServer, orderStreamServer)

	return &App{
		Router:      router,
		GRPCServer:  grpcServer,
		DB:          db,
		RedisClient: redisClient,
	}, nil
}

func newRedisClient() (*redis.Client, error) {
	host := getEnv("REDIS_HOST", "localhost")
	port := getEnv("REDIS_PORT", "6379")
	password := os.Getenv("REDIS_PASSWORD")
	db := parseInt(os.Getenv("REDIS_DB"), 0)

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})
	return client, nil
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
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}
