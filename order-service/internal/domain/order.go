package domain

import (
	"errors"
	"time"
)

type Order struct {
	ID         string
	CustomerID string
	ItemName   string
	Amount     int64 // Amount in cents (e.g., 1000 = $10.00)
	Status     string
	CreatedAt  time.Time
}

const (
	StatusPending   = "Pending"
	StatusPaid      = "Paid"
	StatusFailed    = "Failed"
	StatusCancelled = "Cancelled"
)

var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrInvalidAmount       = errors.New("amount must be greater than 0")
	ErrCannotCancelPaid    = errors.New("paid orders cannot be cancelled")
	ErrCannotCancelNonPend = errors.New("only pending orders can be cancelled")
)

func NewOrder(customerID, itemName string, amount int64) (*Order, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	return &Order{
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     StatusPending,
		CreatedAt:  time.Now(),
	}, nil
}

func (o *Order) Cancel() error {
	if o.Status == StatusPaid {
		return ErrCannotCancelPaid
	}
	if o.Status != StatusPending {
		return ErrCannotCancelNonPend
	}
	o.Status = StatusCancelled
	return nil
}
