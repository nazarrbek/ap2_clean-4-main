package grpc

import (
	"context"
	"fmt"
	"time"

	paymentpb "payment-contract/payment"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpcstatus "google.golang.org/grpc/status"

	"order-service/internal/usecase"
)

type PaymentGRPCClient struct {
	client  paymentpb.PaymentServiceClient
	timeout time.Duration
}

func NewPaymentGRPCClient(addr string, timeout time.Duration) (*PaymentGRPCClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect payment grpc: %w", err)
	}

	return &PaymentGRPCClient{
		client:  paymentpb.NewPaymentServiceClient(conn),
		timeout: timeout,
	}, nil
}

func (c *PaymentGRPCClient) Authorize(ctx context.Context, orderID string, amount int64) (string, string, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ProcessPayment(callCtx, &paymentpb.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		st, ok := grpcstatus.FromError(err)
		if !ok {
			return "", "", usecase.ErrPaymentUnavailable
		}
		if st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded {
			return "", "", usecase.ErrPaymentUnavailable
		}
		return "", "", err
	}

	if !resp.GetSuccess() {
		return resp.GetTransactionId(), resp.GetStatus(), nil
	}

	return resp.GetTransactionId(), resp.GetStatus(), nil
}
