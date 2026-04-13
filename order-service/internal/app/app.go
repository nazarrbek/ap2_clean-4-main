package app

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	orderpb "order-contract/order"

	"order-service/internal/repository"
	ordergrpc "order-service/internal/transport/grpc"
	orderhttp "order-service/internal/transport/http"
	"order-service/internal/usecase"
)

type App struct {
	Router     *gin.Engine
	GRPCServer *grpc.Server
	DB         *sql.DB
}

func NewApp(db *sql.DB, paymentGRPCAddr string) (*App, error) {
	// Composition Root: manual dependency injection
	orderRepo := repository.NewPostgresOrderRepository(db)

	paymentClient, err := ordergrpc.NewPaymentGRPCClient(paymentGRPCAddr, 2*time.Second)
	if err != nil {
		return nil, err
	}

	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient)
	handler := orderhttp.NewOrderHandler(orderUC)
	orderStreamServer := ordergrpc.NewOrderGRPCServer(orderUC)

	router := gin.Default()
	handler.RegisterRoutes(router)

	grpcServer := grpc.NewServer()
	orderpb.RegisterOrderServiceServer(grpcServer, orderStreamServer)

	return &App{Router: router, GRPCServer: grpcServer, DB: db}, nil
}
