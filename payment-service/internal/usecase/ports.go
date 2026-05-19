package usecase

import (
	"context"
	"payment-service/internal/domain"
	"payment-service/internal/messaging"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, event messaging.PaymentEvent) error
}
