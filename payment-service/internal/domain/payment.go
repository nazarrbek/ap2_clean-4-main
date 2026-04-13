package domain

import "errors"

type Payment struct {
	ID            string
	OrderID       string
	TransactionID string
	Amount        int64 // Amount in cents
	Status        string
}

const (
	StatusAuthorized = "Authorized"
	StatusDeclined   = "Declined"

	// MaxAmount: if amount > 100000 (i.e. > 1000 units), decline
	MaxAmount = int64(100000)
)

var (
	ErrPaymentNotFound = errors.New("payment not found")
	ErrInvalidAmount   = errors.New("amount must be greater than 0")
)

func NewPayment(orderID string, amount int64) (*Payment, string) {
	status := StatusAuthorized
	if amount > MaxAmount {
		status = StatusDeclined
	}
	return &Payment{
		OrderID: orderID,
		Amount:  amount,
		Status:  status,
	}, status
}
