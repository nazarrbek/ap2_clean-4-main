package grpc

import (
	"context"
	"errors"

	paymentpb "payment-contract/payment"

	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"payment-service/internal/domain"
	"payment-service/internal/usecase"
)

type PaymentGRPCServer struct {
	paymentpb.UnimplementedPaymentServiceServer
	uc *usecase.PaymentUseCase
}

func NewPaymentGRPCServer(uc *usecase.PaymentUseCase) *PaymentGRPCServer {
	return &PaymentGRPCServer{uc: uc}
}

func (s *PaymentGRPCServer) ProcessPayment(ctx context.Context, req *paymentpb.PaymentRequest) (*paymentpb.PaymentResponse, error) {
	if req.GetOrderId() == "" {
		return nil, grpcstatus.Error(codes.InvalidArgument, "order_id is required")
	}

	// CustomerEmail is not in the proto contract (legacy field).
	// The event will be published with an empty email when called via gRPC.
	// Callers that know the email should use the HTTP endpoint instead.
	output, err := s.uc.Authorize(ctx, usecase.AuthorizeInput{
		OrderID:       req.GetOrderId(),
		Amount:        req.GetAmount(),
		CustomerEmail: "unknown@grpc-caller.internal",
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAmount) {
			return nil, grpcstatus.Error(codes.InvalidArgument, err.Error())
		}
		return nil, grpcstatus.Error(codes.Internal, err.Error())
	}

	message := "payment authorized"
	if output.Payment.Status == domain.StatusDeclined {
		message = "payment declined"
	}

	return &paymentpb.PaymentResponse{
		Success:       output.Payment.Status == domain.StatusAuthorized,
		TransactionId: output.Payment.TransactionID,
		Status:        output.Payment.Status,
		Message:       message,
		ProcessedAt:   timestamppb.Now(),
	}, nil
}
