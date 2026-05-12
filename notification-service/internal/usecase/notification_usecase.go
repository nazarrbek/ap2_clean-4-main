package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"notification-service/internal/adapter"
	"notification-service/internal/domain"
	"notification-service/internal/repository"
)

// NotificationUsecase orchestrates idempotency checks, sending, and retry.
// It uses interfaces — it has no knowledge of Redis or RabbitMQ internals.
type NotificationUsecase interface {
	Process(ctx context.Context, event domain.NotificationEvent) error
}

type notificationUsecase struct {
	idempotency repository.IdempotencyRepository
	sender      adapter.EmailSender
	maxRetries  int
}

func NewNotificationUsecase(
	idempotency repository.IdempotencyRepository,
	sender adapter.EmailSender,
	maxRetries int,
) NotificationUsecase {
	return &notificationUsecase{
		idempotency: idempotency,
		sender:      sender,
		maxRetries:  maxRetries,
	}
}

// Process sends a notification with idempotency guard and exponential backoff retry.
func (uc *notificationUsecase) Process(ctx context.Context, event domain.NotificationEvent) error {
	// ── 1. Idempotency check ─────────────────────────────────────────────────
	// Check if we already processed this payment_id to avoid duplicate emails.
	alreadyProcessed, err := uc.idempotency.IsProcessed(ctx, event.PaymentID)
	if err != nil {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if alreadyProcessed {
		log.Printf("⚡ idempotency: payment %d already processed — skipping", event.PaymentID)
		return nil
	}

	// ── 2. Mark as processing (in-progress) ─────────────────────────────────
	if err := uc.idempotency.MarkProcessing(ctx, event.PaymentID); err != nil {
		return fmt.Errorf("mark processing: %w", err)
	}

	// ── 3. Send with exponential backoff retry ───────────────────────────────
	subject := fmt.Sprintf("Payment Confirmed — Order #%d", event.OrderID)
	body := fmt.Sprintf(
		"Hello! Your payment of $%.2f for Order #%d has been %s.\nTransaction ID: TXN-%d",
		event.Amount, event.OrderID, event.Status, event.PaymentID,
	)

	sendErr := uc.sendWithRetry(ctx, event.CustomerEmail, subject, body)
	if sendErr != nil {
		// Mark as failed so it can be retried on next delivery
		_ = uc.idempotency.MarkFailed(ctx, event.PaymentID)
		return fmt.Errorf("send notification: %w", sendErr)
	}

	// ── 4. Mark as done — future duplicates will be skipped ─────────────────
	if err := uc.idempotency.MarkDone(ctx, event.PaymentID); err != nil {
		log.Printf("warn: mark done payment %d: %v", event.PaymentID, err)
	}

	log.Printf("✅ notification sent to %s for payment %d", event.CustomerEmail, event.PaymentID)
	return nil
}

// sendWithRetry retries with exponential backoff: 2s → 4s → 8s.
func (uc *notificationUsecase) sendWithRetry(ctx context.Context, to, subject, body string) error {
	var lastErr error
	for attempt := 1; attempt <= uc.maxRetries; attempt++ {
		err := uc.sender.Send(ctx, to, subject, body)
		if err == nil {
			return nil
		}
		lastErr = err
		backoff := time.Duration(1<<attempt) * time.Second // 2s, 4s, 8s
		log.Printf("⚠️  attempt %d/%d failed: %v — retrying in %s", attempt, uc.maxRetries, err, backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
	return fmt.Errorf("all %d attempts failed: %w", uc.maxRetries, lastErr)
}
