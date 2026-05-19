package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeName = "payments"
	routingKey   = "payment.completed"
)

// PaymentEvent is the message schema published to the broker.
type PaymentEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

// Publisher publishes payment events to RabbitMQ.
type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewPublisher dials RabbitMQ and declares the exchange.
func NewPublisher(amqpURL string) (*Publisher, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	// Declare durable exchange so it survives broker restarts
	if err := ch.ExchangeDeclare(
		exchangeName,
		"direct",
		true,  // durable
		false, false, false, nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	// Enable publisher confirms for reliability
	if err := ch.Confirm(false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("enable confirms: %w", err)
	}

	log.Println("[Publisher] RabbitMQ publisher ready")
	return &Publisher{conn: conn, channel: ch}, nil
}

// Publish sends a PaymentEvent to the broker and waits for broker confirmation.
func (p *Publisher) Publish(ctx context.Context, event PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	confirms := p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	err = p.channel.PublishWithContext(ctx,
		exchangeName,
		routingKey,
		false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // survive broker restart
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	// Wait for broker ACK (publisher confirm)
	select {
	case confirm := <-confirms:
		if !confirm.Ack {
			return fmt.Errorf("broker nacked message for event %s", event.EventID)
		}
		log.Printf("[Publisher] Event %s confirmed by broker", event.EventID)
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// Close gracefully closes channel and connection.
func (p *Publisher) Close() {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
	log.Println("[Publisher] RabbitMQ connection closed.")
}
