package domain

// PaymentEvent is the message published by the Payment Service
// and consumed by the Notification Service.
type PaymentEvent struct {
	EventID       string `json:"event_id"`       // Unique ID for idempotency (RabbitMQ dedup)
	PaymentID     string `json:"payment_id"`     // Payment ID used for Redis idempotency key
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`         // in cents
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}
