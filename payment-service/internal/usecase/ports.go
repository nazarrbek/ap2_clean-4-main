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

// EventPublisher is the messaging port. The usecase depends on this interface,
// not on RabbitMQ directly (Separation of Concerns / Clean Architecture).
type EventPublisher interface {
	Publish(ctx context.Context, event messaging.PaymentEvent) error
}
