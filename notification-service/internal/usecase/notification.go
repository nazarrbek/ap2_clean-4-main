package usecase

import (
	"context"
	"fmt"
	"log"
	"notification-service/internal/domain"
	"notification-service/internal/provider"
	"notification-service/internal/retry"
)

// IdempotencyStore checks and records processed payment IDs.
// Implemented by idempotency.RedisIdempotencyStore.
type IdempotencyStore interface {
	IsProcessed(ctx context.Context, paymentID string) (bool, error)
	MarkProcessed(ctx context.Context, paymentID string) error
}

// NotificationUseCase handles the business logic for processing payment events.
// It is completely decoupled from Redis and SMTP implementation details.
type NotificationUseCase struct {
	emailSender  provider.EmailSender
	idempotency  IdempotencyStore
	retryConfig  retry.Config
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

// Handle processes a payment event asynchronously.
// Returns (true, nil) when notification was sent.
// Returns (false, nil) when event was a duplicate (already processed).
// Returns (false, err) when all retry attempts are exhausted.
func (uc *NotificationUseCase) Handle(ctx context.Context, event domain.PaymentEvent) (sent bool, err error) {
	// 1. Idempotency check: has this payment been processed before?
	already, err := uc.idempotency.IsProcessed(ctx, event.PaymentID)
	if err != nil {
		log.Printf("[Notification] Failed to check idempotency for payment %s: %v", event.PaymentID, err)
		// Don't block on Redis error – proceed (may cause a duplicate, acceptable trade-off)
	}
	if already {
		log.Printf("[Notification] DUPLICATE payment %s – skipping", event.PaymentID)
		return false, nil
	}

	// 2. Build the email message
	msg := provider.EmailMessage{
		To:      event.CustomerEmail,
		Subject: fmt.Sprintf("Order #%s – Payment Confirmed", event.OrderID),
		Body: fmt.Sprintf(
			"Hello!\n\nYour payment of $%.2f for Order #%s has been successfully processed.\n\nThank you for your purchase!\n",
			float64(event.Amount)/100.0, event.OrderID,
		),
	}

	// 3. Send with exponential backoff retries
	jobName := fmt.Sprintf("send-email[payment=%s]", event.PaymentID)
	sendErr := retry.Do(ctx, uc.retryConfig, jobName, func() error {
		return uc.emailSender.Send(ctx, msg)
	})

	if sendErr != nil {
		return false, fmt.Errorf("all retry attempts failed: %w", sendErr)
	}

	// 4. Mark as processed ONLY after successful send (prevents double-send on retries)
	if err := uc.idempotency.MarkProcessed(ctx, event.PaymentID); err != nil {
		log.Printf("[Notification] Warning: failed to mark payment %s as processed in Redis: %v", event.PaymentID, err)
		// Continue – the email was sent successfully; Redis failure is non-fatal here
	}

	return true, nil
}
