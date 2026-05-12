package domain

import (
	"time"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusPaid      OrderStatus = "paid"
	StatusCancelled OrderStatus = "cancelled"
)

// Order is the core domain entity.
type Order struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	CustomerID  string      `gorm:"not null"   json:"customer_id"`
	ItemName    string      `gorm:"not null"   json:"item_name"`
	Amount      float64     `gorm:"not null"   json:"amount"`
	Status      OrderStatus `gorm:"default:pending" json:"status"`
	PaymentID   string      `json:"payment_id,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}
