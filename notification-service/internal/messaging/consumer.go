package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"notification-service/internal/domain"
	"notification-service/internal/usecase"
)

const (
	exchangeName = "payments"
	queueName    = "payment.completed"
	dlxName      = "payments.dlx"
	dlqName      = "payment.completed.dlq"
	routingKey   = "payment.completed"
)

// Consumer listens to the payment.completed queue.
type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	uc      *usecase.NotificationUseCase
}

// NewConsumer dials RabbitMQ and sets up all topology (exchange, DLX, DLQ, queue).
func NewConsumer(amqpURL string, uc *usecase.NotificationUseCase) (*Consumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	if err := declareToplogy(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	// Process one message at a time (fair dispatch)
	if err := ch.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("set qos: %w", err)
	}

	return &Consumer{conn: conn, channel: ch, uc: uc}, nil
}

func declareToplogy(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(exchangeName, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(dlxName, "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx: %w", err)
	}
	if _, err := ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlq: %w", err)
	}
	if err := ch.QueueBind(dlqName, "", dlxName, false, nil); err != nil {
		return fmt.Errorf("bind dlq: %w", err)
	}

	args := amqp.Table{"x-dead-letter-exchange": dlxName}
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	if err := ch.QueueBind(queueName, routingKey, exchangeName, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}
	return nil
}

// Run starts consuming messages until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		queueName,
		"",    // consumer tag
		false, // auto-ack = false  ← manual ACK
		false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Printf("[Consumer] Listening on queue %q", queueName)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Consumer] Context cancelled, stopping.")
			return nil

		case msg, ok := <-msgs:
			if !ok {
				log.Println("[Consumer] Channel closed.")
				return nil
			}
			c.handleDelivery(ctx, msg)
		}
	}
}

// handleDelivery processes a single RabbitMQ message.
// Retry logic with exponential backoff is handled INSIDE the use-case layer.
// If all retries are exhausted, we NACK (→ DLQ). If duplicate, we ACK silently.
func (c *Consumer) handleDelivery(ctx context.Context, msg amqp.Delivery) {
	var event domain.PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[Consumer] Bad payload: %v – sending to DLQ", err)
		_ = msg.Nack(false, false) // malformed → DLQ
		return
	}

	// Fallback: use EventID as PaymentID if not set (backwards compat)
	if event.PaymentID == "" {
		event.PaymentID = event.EventID
	}

	sent, err := c.uc.Handle(ctx, event)
	if err != nil {
		log.Printf("[Consumer] All retries exhausted for payment %s: %v – sending to DLQ",
			event.PaymentID, err)
		_ = msg.Nack(false, false) // → DLQ for manual inspection
		return
	}

	if !sent {
		log.Printf("[Consumer] Duplicate event for payment %s – ACK without action", event.PaymentID)
	} else {
		log.Printf("[Consumer] ✓ Notification sent for Order #%s (payment %s, $%.2f to %s)",
			event.OrderID, event.PaymentID, float64(event.Amount)/100.0, event.CustomerEmail)
	}

	// ACK only after successful processing (at-least-once semantics)
	_ = msg.Ack(false)
}

// Close gracefully closes channel and connection.
func (c *Consumer) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	log.Println("[Consumer] RabbitMQ connection closed.")
}
