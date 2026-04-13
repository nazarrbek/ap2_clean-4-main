package usecase

import (
	"context"
	"errors"
	"order-service/internal/domain"
)

type OrderUseCase struct {
	repo          OrderRepository
	paymentClient PaymentClient
}

func NewOrderUseCase(repo OrderRepository, paymentClient PaymentClient) *OrderUseCase {
	return &OrderUseCase{repo: repo, paymentClient: paymentClient}
}

type CreateOrderInput struct {
	CustomerID     string
	ItemName       string
	Amount         int64
	IdempotencyKey string
}

type CreateOrderOutput struct {
	Order *domain.Order
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, input CreateOrderInput) (*CreateOrderOutput, error) {
	// Idempotency check
	if input.IdempotencyKey != "" {
		existing, err := uc.repo.GetByIdempotencyKey(ctx, input.IdempotencyKey)
		if err == nil && existing != nil {
			return &CreateOrderOutput{Order: existing}, nil
		}
	}

	order, err := domain.NewOrder(input.CustomerID, input.ItemName, input.Amount)
	if err != nil {
		return nil, err
	}

	// Generate ID
	order.ID = generateID()

	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	// Save idempotency key if provided
	if input.IdempotencyKey != "" {
		_ = uc.repo.SaveIdempotencyKey(ctx, input.IdempotencyKey, order.ID)
	}

	// Call Payment Service
	_, status, err := uc.paymentClient.Authorize(ctx, order.ID, order.Amount)
	if err != nil {
		// Payment service unavailable or timeout
		// Mark as Failed
		_ = uc.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed)
		order.Status = domain.StatusFailed
		return &CreateOrderOutput{Order: order}, err
	}

	newStatus := domain.StatusFailed
	if status == "Authorized" {
		newStatus = domain.StatusPaid
	}

	if err := uc.repo.UpdateStatus(ctx, order.ID, newStatus); err != nil {
		return nil, err
	}
	order.Status = newStatus

	return &CreateOrderOutput{Order: order}, nil
}

func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *OrderUseCase) GetRecentOrders(ctx context.Context, limit int) ([]*domain.Order, error) {
	if limit <= 0 || limit > 100 {
		limit = 5
	}
	return uc.repo.GetRecent(ctx, limit)
}

func (uc *OrderUseCase) CancelOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := order.Cancel(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateStatus(ctx, id, domain.StatusCancelled); err != nil {
		return nil, err
	}

	return order, nil
}

// ErrPaymentUnavailable is returned when Payment Service cannot be reached
var ErrPaymentUnavailable = errors.New("payment service unavailable")

func generateID() string {
	// Simple UUID-like generation using crypto/rand
	b := make([]byte, 16)
	_, _ = cryptoRead(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return formatUUID(b)
}
