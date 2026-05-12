package repository

import (
	"context"

	"gorm.io/gorm"

	"order-service/internal/domain"
)

// OrderRepository defines persistence operations.
type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	FindByID(ctx context.Context, id uint) (*domain.Order, error)
	UpdateStatus(ctx context.Context, id uint, status domain.OrderStatus, paymentID string) error
	ListRecent(ctx context.Context, limit int) ([]domain.Order, error)
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, order *domain.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *orderRepository) FindByID(ctx context.Context, id uint) (*domain.Order, error) {
	var order domain.Order
	if err := r.db.WithContext(ctx).First(&order, id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id uint, status domain.OrderStatus, paymentID string) error {
	return r.db.WithContext(ctx).Model(&domain.Order{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"payment_id": paymentID,
		}).Error
}

func (r *orderRepository) ListRecent(ctx context.Context, limit int) ([]domain.Order, error) {
	var orders []domain.Order
	if err := r.db.WithContext(ctx).Order("created_at desc").Limit(limit).Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}
