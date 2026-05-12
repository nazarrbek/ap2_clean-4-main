package domain

import "time"

type PaymentStatus string

const (
	PaymentStatusAuthorized PaymentStatus = "authorized"
	PaymentStatusFailed     PaymentStatus = "failed"
)

// Payment is the core domain entity for payment-service.
type Payment struct {
	ID            uint          `gorm:"primaryKey"  json:"id"`
	OrderID       uint          `gorm:"not null"    json:"order_id"`
	CustomerEmail string        `gorm:"not null"    json:"customer_email"`
	Amount        float64       `gorm:"not null"    json:"amount"`
	Status        PaymentStatus `gorm:"default:authorized" json:"status"`
	TransactionID string        `json:"transaction_id"`
	CreatedAt     time.Time     `json:"created_at"`
}

// PaymentEvent is the message published to RabbitMQ.
type PaymentEvent struct {
	PaymentID     uint    `json:"payment_id"`
	OrderID       uint    `json:"order_id"`
	CustomerEmail string  `json:"customer_email"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
}
