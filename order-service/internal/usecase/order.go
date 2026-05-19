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

	order.ID = generateID()

	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	if input.IdempotencyKey != "" {
		_ = uc.repo.SaveIdempotencyKey(ctx, input.IdempotencyKey, order.ID)
	}

	// Call Payment Service
	_, status, err := uc.paymentClient.Authorize(ctx, order.ID, order.Amount)
	if err != nil {
		_ = uc.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed)
		order.Status = domain.StatusFailed
		// Invalidate cache (order may have been cached with old status)
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

	// Atomic invalidation: delete cache after DB update so next read is fresh.
	uc.invalidateCache(ctx, order.ID)

	return &CreateOrderOutput{Order: order}, nil
}

// GetOrder implements cache-aside: check Redis first, fallback to DB, then populate cache.
func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	// 1. Check cache
	if cached, err := uc.cache.Get(ctx, id); err == nil && cached != nil {
		log.Printf("[Cache] HIT for order %s", id)
		return cached, nil
	}

	log.Printf("[Cache] MISS for order %s – fetching from DB", id)

	// 2. Fallback to DB
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Populate cache (best-effort, don't fail the request on cache write error)
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

	// Invalidate cache immediately after DB update
	uc.invalidateCache(ctx, id)

	order.Status = domain.StatusCancelled
	return order, nil
}

// invalidateCache deletes an order from Redis. Errors are logged but not returned
// because a cache error should never cause a business operation to fail.
func (uc *OrderUseCase) invalidateCache(ctx context.Context, id string) {
	if err := uc.cache.Delete(ctx, id); err != nil {
		log.Printf("[Cache] Failed to invalidate cache for order %s: %v", id, err)
	} else {
		log.Printf("[Cache] Invalidated cache for order %s", id)
	}
}

// ErrPaymentUnavailable is returned when Payment Service cannot be reached.
var ErrPaymentUnavailable = errors.New("payment service unavailable")

func generateID() string {
	b := make([]byte, 16)
	_, _ = cryptoRead(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return formatUUID(b)
}
