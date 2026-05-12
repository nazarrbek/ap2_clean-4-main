package domain

// NotificationEvent is received from RabbitMQ.
type NotificationEvent struct {
	PaymentID     uint    `json:"payment_id"`
	OrderID       uint    `json:"order_id"`
	CustomerEmail string  `json:"customer_email"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
}
