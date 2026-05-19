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

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	uc      *usecase.NotificationUseCase
}

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

func (c *Consumer) Run(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		queueName,
		"",
		false,
		false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Printf("[Consumer] Listening on queue %q", queueName)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Consumer] context cancelled, stopping")
			return nil

		case msg, ok := <-msgs:
			if !ok {
				log.Println("[Consumer] channel closed")
				return nil
			}
			c.handleDelivery(ctx, msg)
		}
	}
}

func (c *Consumer) handleDelivery(ctx context.Context, msg amqp.Delivery) {
	var event domain.PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[Consumer] bad payload: %v", err)
		_ = msg.Nack(false, false)
		return
	}

	if event.PaymentID == "" {
		event.PaymentID = event.EventID
	}

	sent, err := c.uc.Handle(ctx, event)
	if err != nil {
		log.Printf("[Consumer] retries exhausted for payment %s: %v",
			event.PaymentID, err)
		_ = msg.Nack(false, false)
		return
	}

	if !sent {
		log.Printf("[Consumer] duplicate event for payment %s", event.PaymentID)
	} else {
		log.Printf("[Consumer] notification sent for order %s (payment %s, $%.2f to %s)",
			event.OrderID, event.PaymentID, float64(event.Amount)/100.0, event.CustomerEmail)
	}

	_ = msg.Ack(false)
}

func (c *Consumer) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	log.Println("[Consumer] RabbitMQ connection closed")
}
