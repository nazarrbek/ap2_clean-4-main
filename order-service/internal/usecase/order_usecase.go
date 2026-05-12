package usecase

import (
	"context"
	"fmt"
	"log"

	"order-service/internal/cache"
	"order-service/internal/domain"
	"order-service/internal/repository"
)

// OrderUsecase contains the business logic.
// It knows nothing about Redis or HTTP — it uses interfaces.
type OrderUsecase interface {
	CreateOrder(ctx context.Context, customerID, itemName string, amount float64) (*domain.Order, error)
	GetOrder(ctx context.Context, id uint) (*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, id uint, status domain.OrderStatus, paymentID string) (*domain.Order, error)
	ListRecent(ctx context.Context) ([]domain.Order, error)
}

type orderUsecase struct {
	repo  repository.OrderRepository
	cache cache.OrderCache
}

func NewOrderUsecase(repo repository.OrderRepository, cache cache.OrderCache) OrderUsecase {
	return &orderUsecase{repo: repo, cache: cache}
}

func (uc *orderUsecase) CreateOrder(ctx context.Context, customerID, itemName string, amount float64) (*domain.Order, error) {
	if customerID == "" || itemName == "" || amount <= 0 {
		return nil, fmt.Errorf("invalid order fields")
	}
	order := &domain.Order{
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     domain.StatusPending,
	}
	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	// Populate cache right away (optional warm-up)
	if err := uc.cache.Set(ctx, order); err != nil {
		log.Printf("warn: cache set after create: %v", err)
	}
	return order, nil
}

// GetOrder implements cache-aside:
//  1. Check Redis.
//  2. On miss → query DB → store result in Redis.
func (uc *orderUsecase) GetOrder(ctx context.Context, id uint) (*domain.Order, error) {
	// 1. Cache read
	order, err := uc.cache.Get(ctx, id)
	if err != nil {
		log.Printf("warn: cache get order %d: %v", id, err)
	}
	if order != nil {
		log.Printf("cache HIT for order %d", id)
		return order, nil
	}

	// 2. Cache miss → DB read
	log.Printf("cache MISS for order %d – querying DB", id)
	order, err = uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find order %d: %w", id, err)
	}

	// 3. Populate cache
	if err := uc.cache.Set(ctx, order); err != nil {
		log.Printf("warn: cache set order %d: %v", id, err)
	}
	return order, nil
}

// UpdateOrderStatus changes status in DB and invalidates the cache atomically.
func (uc *orderUsecase) UpdateOrderStatus(ctx context.Context, id uint, status domain.OrderStatus, paymentID string) (*domain.Order, error) {
	// 1. DB update
	if err := uc.repo.UpdateStatus(ctx, id, status, paymentID); err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}

	// 2. Cache invalidation — prevent stale reads
	if err := uc.cache.Delete(ctx, id); err != nil {
		log.Printf("warn: cache delete order %d: %v", id, err)
	}

	// 3. Return fresh data from DB
	return uc.repo.FindByID(ctx, id)
}

func (uc *orderUsecase) ListRecent(ctx context.Context) ([]domain.Order, error) {
	return uc.repo.ListRecent(ctx, 20)
}
