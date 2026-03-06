// Package rabbitmq provides a RabbitMQ implementation of the core.Broker interface.
package rabbitmq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"agents/internal/core"

	amqp "github.com/rabbitmq/amqp091-go"
)

// rmqEvent implements the core.Event interface for RabbitMQ deliveries.
type rmqEvent struct {
	d amqp.Delivery
}

func (e *rmqEvent) Payload() []byte { return e.d.Body }
func (e *rmqEvent) Ack() error      { return e.d.Ack(false) }
func (e *rmqEvent) Nack() error     { return e.d.Nack(false, false) }

// RabbitMQBroker implements the core.Broker interface.
type RabbitMQBroker struct {
	url string

	mu   sync.RWMutex
	conn *amqp.Connection
	ch   *amqp.Channel
}

// NewBroker initializes a new RabbitMQ broker configuration.
func NewBroker(url string) *RabbitMQBroker {
	return &RabbitMQBroker{url: url}
}

// Connect establishes a connection and opens a channel to the RabbitMQ server.
func (r *RabbitMQBroker) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, err := amqp.Dial(r.url)
	if err != nil {
		return fmt.Errorf("failed to dial rabbitmq: %w", err)
	}
	r.conn = conn

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to open rabbitmq channel: %w", err)
	}
	r.ch = ch

	return nil
}

// Close gracefully shuts down the channel and the connection.
func (r *RabbitMQBroker) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ch != nil {
		_ = r.ch.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// Publish sends a message to a specific queue using the default exchange.
func (r *RabbitMQBroker) Publish(ctx context.Context, queue string, msg []byte) error {
	r.mu.RLock()
	ch := r.ch
	r.mu.RUnlock()

	if ch == nil {
		return fmt.Errorf("broker not connected")
	}

	// Ensure a timeout exists for the operation
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	err := ch.PublishWithContext(ctx,
		"",    // exchange (default)
		queue, // routing key (queue name)
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msg,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish to queue %s: %w", queue, err)
	}
	return nil
}

// Subscribe listens for messages on a specific queue and returns them as a channel of core.Events.
// If the queue does not exist, it is declared as a durable queue.
func (r *RabbitMQBroker) Subscribe(ctx context.Context, queue string) (<-chan core.Event, error) {
	r.mu.RLock()
	ch := r.ch
	r.mu.RUnlock()

	if ch == nil {
		return nil, fmt.Errorf("broker not connected")
	}

	// Ensure the queue exists
	_, err := ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue %s: %w", queue, err)
	}

	msgs, err := ch.Consume(
		queue,
		"",    // consumer name
		false, // auto-ack (manual Ack required)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming from %s: %w", queue, err)
	}

	out := make(chan core.Event)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-msgs:
				if !ok {
					return
				}
				out <- &rmqEvent{d: d}
			}
		}
	}()
	return out, nil
}
