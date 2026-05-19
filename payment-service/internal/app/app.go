package app

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	paymentpb "payment-contract/payment"
	"payment-service/internal/messaging"
	"payment-service/internal/repository"
	transportgrpc "payment-service/internal/transport/grpc"
	transporthttp "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
)

type App struct {
	Router     *gin.Engine
	GRPCServer *grpc.Server
	DB         *sql.DB
	Publisher  *messaging.Publisher
}

func NewApp(db *sql.DB, amqpURL string) (*App, error) {
	pub, err := messaging.NewPublisher(amqpURL)
	if err != nil {
		return nil, err
	}

	paymentRepo := repository.NewPostgresPaymentRepository(db)
	paymentUC := usecase.NewPaymentUseCase(paymentRepo, pub)
	handler := transporthttp.NewPaymentHandler(paymentUC)
	grpcHandler := transportgrpc.NewPaymentGRPCServer(paymentUC)

	router := gin.Default()
	handler.RegisterRoutes(router)

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		start := time.Now()
		resp, err = handler(ctx, req)
		log.Printf("grpc method=%s duration=%s err=%v", info.FullMethod, time.Since(start), err)
		return resp, err
	}))
	paymentpb.RegisterPaymentServiceServer(grpcServer, grpcHandler)

	return &App{Router: router, GRPCServer: grpcServer, DB: db, Publisher: pub}, nil
}
