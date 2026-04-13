package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"payment-service/internal/domain"
)

type PaymentUseCase struct {
	repo PaymentRepository
}

func NewPaymentUseCase(repo PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repo: repo}
}

type AuthorizeInput struct {
	OrderID string
	Amount  int64
}

type AuthorizeOutput struct {
	Payment *domain.Payment
}

func (uc *PaymentUseCase) Authorize(ctx context.Context, input AuthorizeInput) (*AuthorizeOutput, error) {
	if input.Amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}

	payment, _ := domain.NewPayment(input.OrderID, input.Amount)
	payment.ID = generateID()
	payment.TransactionID = generateTransactionID()

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, err
	}

	return &AuthorizeOutput{Payment: payment}, nil
}

func (uc *PaymentUseCase) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return uc.repo.GetByOrderID(ctx, orderID)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateTransactionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("TXN-%x", b)
}
