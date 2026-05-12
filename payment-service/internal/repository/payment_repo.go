package repository

import (
	"context"

	"gorm.io/gorm"

	"payment-service/internal/domain"
)

type PaymentRepository interface {
	Create(ctx context.Context, p *domain.Payment) error
	FindByID(ctx context.Context, id uint) (*domain.Payment, error)
	FindByOrderID(ctx context.Context, orderID uint) (*domain.Payment, error)
}

type paymentRepository struct{ db *gorm.DB }

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *paymentRepository) FindByID(ctx context.Context, id uint) (*domain.Payment, error) {
	var p domain.Payment
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepository) FindByOrderID(ctx context.Context, orderID uint) (*domain.Payment, error) {
	var p domain.Payment
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}
