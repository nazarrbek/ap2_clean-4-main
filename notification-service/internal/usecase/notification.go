package usecase

import (
	"context"
	"fmt"
	"log"
	"notification-service/internal/domain"
	"notification-service/internal/provider"
	"notification-service/internal/retry"
)

type IdempotencyStore interface {
	IsProcessed(ctx context.Context, paymentID string) (bool, error)
	MarkProcessed(ctx context.Context, paymentID string) error
}

type NotificationUseCase struct {
	emailSender provider.EmailSender
	idempotency IdempotencyStore
	retryConfig retry.Config
}

func NewNotificationUseCase(
	emailSender provider.EmailSender,
	idempotency IdempotencyStore,
	retryCfg retry.Config,
) *NotificationUseCase {
	return &NotificationUseCase{
		emailSender: emailSender,
		idempotency: idempotency,
		retryConfig: retryCfg,
	}
}

func (uc *NotificationUseCase) Handle(ctx context.Context, event domain.PaymentEvent) (sent bool, err error) {
	already, err := uc.idempotency.IsProcessed(ctx, event.PaymentID)
	if err != nil {
		log.Printf("[Notification] Failed to check idempotency for payment %s: %v", event.PaymentID, err)
	}
	if already {
		log.Printf("[Notification] DUPLICATE payment %s – skipping", event.PaymentID)
		return false, nil
	}

	msg := provider.EmailMessage{
		To:      event.CustomerEmail,
		Subject: fmt.Sprintf("Order #%s – Payment Confirmed", event.OrderID),
		Body: fmt.Sprintf(
			"Hello!\n\nYour payment of $%.2f for Order #%s has been successfully processed.\n\nThank you for your purchase!\n",
			float64(event.Amount)/100.0, event.OrderID,
		),
	}

	jobName := fmt.Sprintf("send-email[payment=%s]", event.PaymentID)
	sendErr := retry.Do(ctx, uc.retryConfig, jobName, func() error {
		return uc.emailSender.Send(ctx, msg)
	})

	if sendErr != nil {
		return false, fmt.Errorf("all retry attempts failed: %w", sendErr)
	}

	if err := uc.idempotency.MarkProcessed(ctx, event.PaymentID); err != nil {
		log.Printf("[Notification] Warning: failed to mark payment %s as processed in Redis: %v", event.PaymentID, err)
	}

	return true, nil
}
