package grpc

import (
	"time"

	orderpb "order-contract/order"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"order-service/internal/domain"
	"order-service/internal/usecase"
)

type OrderGRPCServer struct {
	orderpb.UnimplementedOrderServiceServer
	uc *usecase.OrderUseCase
}

func NewOrderGRPCServer(uc *usecase.OrderUseCase) *OrderGRPCServer {
	return &OrderGRPCServer{uc: uc}
}

func (s *OrderGRPCServer) SubscribeToOrderUpdates(req *orderpb.OrderRequest, stream grpc.ServerStreamingServer[orderpb.OrderStatusUpdate]) error {
	if req.GetOrderId() == "" {
		return grpcstatus.Error(codes.InvalidArgument, "order_id is required")
	}

	ctx := stream.Context()
	orderEntity, err := s.uc.GetOrder(ctx, req.GetOrderId())
	if err != nil {
		if err == domain.ErrOrderNotFound {
			return grpcstatus.Error(codes.NotFound, "order not found")
		}
		return grpcstatus.Error(codes.Internal, err.Error())
	}

	lastStatus := orderEntity.Status
	if err := stream.Send(&orderpb.OrderStatusUpdate{
		OrderId:   orderEntity.ID,
		Status:    orderEntity.Status,
		CreatedAt: timestamppb.New(orderEntity.CreatedAt),
		EmittedAt: timestamppb.Now(),
	}); err != nil {
		return err
	}

	ticker := time.NewTicker(800 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			currentOrder, err := s.uc.GetOrder(ctx, req.GetOrderId())
			if err != nil {
				if err == domain.ErrOrderNotFound {
					return grpcstatus.Error(codes.NotFound, "order not found")
				}
				return grpcstatus.Error(codes.Internal, err.Error())
			}

			if currentOrder.Status == lastStatus {
				continue
			}

			lastStatus = currentOrder.Status
			if err := stream.Send(&orderpb.OrderStatusUpdate{
				OrderId:   currentOrder.ID,
				Status:    currentOrder.Status,
				CreatedAt: timestamppb.New(currentOrder.CreatedAt),
				EmittedAt: timestamppb.Now(),
			}); err != nil {
				return err
			}
		}
	}
}
