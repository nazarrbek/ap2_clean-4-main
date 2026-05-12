package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"payment-service/internal/domain"
)

const (
	exchangeName = "payment_events"
	routingKey   = "payment.processed"
)

// Publisher is responsible for publishing payment events to RabbitMQ.
type Publisher interface {
	Publish(ctx context.Context, event domain.PaymentEvent) error
}

type rabbitPublisher struct {
	channel *amqp.Channel
}

func NewPublisher(conn *amqp.Connection) (Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	// Declare a durable topic exchange
	if err := ch.ExchangeDeclare(
		exchangeName, "topic", true, false, false, false, nil,
	); err != nil {
		return nil, fmt.Errorf("declare exchange: %w", err)
	}
	return &rabbitPublisher{channel: ch}, nil
}

func (p *rabbitPublisher) Publish(ctx context.Context, event domain.PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.channel.PublishWithContext(ctx,
		exchangeName,
		routingKey,
		false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}
