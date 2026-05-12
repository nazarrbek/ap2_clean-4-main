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
	exchangeName = "payment_events"
	queueName    = "notifications"
	routingKey   = "payment.processed"
)

// Consumer is the RabbitMQ background worker for the Notification Service.
type Consumer struct {
	conn    *amqp.Connection
	usecase usecase.NotificationUsecase
}

func NewConsumer(conn *amqp.Connection, uc usecase.NotificationUsecase) *Consumer {
	return &Consumer{conn: conn, usecase: uc}
}

// Start begins consuming messages from RabbitMQ. Blocks until ctx is done.
func (c *Consumer) Start(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	// Declare exchange (idempotent — matches publisher)
	if err := ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	// Declare durable queue
	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	// Bind queue to exchange
	if err := ch.QueueBind(q.Name, routingKey, exchangeName, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}

	// Set prefetch to 1 for fair dispatch
	_ = ch.Qos(1, 0, false)

	msgs, err := ch.Consume(q.Name, "", false /*autoAck*/, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Println("🔔 Notification worker started — waiting for events...")

	for {
		select {
		case <-ctx.Done():
			log.Println("notification worker shutting down")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("channel closed unexpectedly")
			}
			c.handleMessage(ctx, msg)
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	var event domain.NotificationEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("❌ failed to unmarshal event: %v — rejecting", err)
		_ = msg.Nack(false, false) // send to DLQ
		return
	}

	log.Printf("📩 received payment event: payment_id=%d order_id=%d email=%s",
		event.PaymentID, event.OrderID, event.CustomerEmail)

	if err := c.usecase.Process(ctx, event); err != nil {
		log.Printf("❌ process event failed: %v — nacking", err)
		_ = msg.Nack(false, false) // RabbitMQ will redeliver
		return
	}

	// ACK only after successful processing (at-least-once guarantee)
	_ = msg.Ack(false)
}
