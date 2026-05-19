package usecase

import (
	"context"
	"errors"
	"log"
	"order-service/internal/domain"
)

type OrderUseCase struct {
	repo          OrderRepository
	cache         OrderCache
	paymentClient PaymentClient
}

func NewOrderUseCase(repo OrderRepository, cache OrderCache, paymentClient PaymentClient) *OrderUseCase {
	return &OrderUseCase{repo: repo, cache: cache, paymentClient: paymentClient}
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

	order.ID = generateID()

	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	if input.IdempotencyKey != "" {
		_ = uc.repo.SaveIdempotencyKey(ctx, input.IdempotencyKey, order.ID)
	}

	_, status, err := uc.paymentClient.Authorize(ctx, order.ID, order.Amount)
	if err != nil {
		_ = uc.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed)
		order.Status = domain.StatusFailed
		uc.invalidateCache(ctx, order.ID)
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

	uc.invalidateCache(ctx, order.ID)

	return &CreateOrderOutput{Order: order}, nil
}

func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	if cached, err := uc.cache.Get(ctx, id); err == nil && cached != nil {
		log.Printf("[Cache] HIT for order %s", id)
		return cached, nil
	}

	log.Printf("[Cache] MISS for order %s – fetching from DB", id)

	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := uc.cache.Set(ctx, order); err != nil {
		log.Printf("[Cache] Failed to set cache for order %s: %v", id, err)
	}

	return order, nil
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

	uc.invalidateCache(ctx, id)

	order.Status = domain.StatusCancelled
	return order, nil
}

func (uc *OrderUseCase) invalidateCache(ctx context.Context, id string) {
	if err := uc.cache.Delete(ctx, id); err != nil {
		log.Printf("[Cache] Failed to invalidate cache for order %s: %v", id, err)
	} else {
		log.Printf("[Cache] Invalidated cache for order %s", id)
	}
}

var ErrPaymentUnavailable = errors.New("payment service unavailable")

func generateID() string {
	b := make([]byte, 16)
	_, _ = cryptoRead(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return formatUUID(b)
}
