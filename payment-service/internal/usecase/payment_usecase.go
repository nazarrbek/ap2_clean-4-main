package usecase

import (
	"context"
	"fmt"

	"payment-service/internal/domain"
	"payment-service/internal/messaging"
	"payment-service/internal/repository"
)

type PaymentUsecase interface {
	ProcessPayment(ctx context.Context, orderID uint, email string, amount float64) (*domain.Payment, error)
	GetPayment(ctx context.Context, id uint) (*domain.Payment, error)
	GetByOrderID(ctx context.Context, orderID uint) (*domain.Payment, error)
}

type paymentUsecase struct {
	repo      repository.PaymentRepository
	publisher messaging.Publisher
}

func NewPaymentUsecase(repo repository.PaymentRepository, publisher messaging.Publisher) PaymentUsecase {
	return &paymentUsecase{repo: repo, publisher: publisher}
}

func (uc *paymentUsecase) ProcessPayment(ctx context.Context, orderID uint, email string, amount float64) (*domain.Payment, error) {
	if orderID == 0 || email == "" || amount <= 0 {
		return nil, fmt.Errorf("invalid payment fields")
	}

	payment := &domain.Payment{
		OrderID:       orderID,
		CustomerEmail: email,
		Amount:        amount,
		Status:        domain.PaymentStatusAuthorized,
		TransactionID: fmt.Sprintf("TXN-%d", orderID),
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	// Publish event to RabbitMQ — notification-service will pick this up
	event := domain.PaymentEvent{
		PaymentID:     payment.ID,
		OrderID:       payment.OrderID,
		CustomerEmail: payment.CustomerEmail,
		Amount:        payment.Amount,
		Status:        string(payment.Status),
	}
	if err := uc.publisher.Publish(ctx, event); err != nil {
		// Log but don't fail the payment — publishing is best-effort at this layer
		fmt.Printf("warn: publish event for payment %d: %v\n", payment.ID, err)
	}

	return payment, nil
}

func (uc *paymentUsecase) GetPayment(ctx context.Context, id uint) (*domain.Payment, error) {
	return uc.repo.FindByID(ctx, id)
}

func (uc *paymentUsecase) GetByOrderID(ctx context.Context, orderID uint) (*domain.Payment, error) {
	return uc.repo.FindByOrderID(ctx, orderID)
}
