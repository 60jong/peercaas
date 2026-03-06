// Package core defines the domain interfaces and shared messaging types for PeerCaaS agents.
package core

import (
	"context"
	"encoding/json"
)

// Broker defines the interface for message queuing operations.
// It allows for connecting to, closing, and performing pub/sub actions on a message broker.
type Broker interface {
	Connect() error
	Close() error
	Publish(ctx context.Context, queue string, body []byte) error
	Subscribe(ctx context.Context, queue string) (<-chan Event, error)
}

// Event represents a single message from the broker.
// It provides methods for accessing the message payload and acknowledging its processing state.
type Event interface {
	Payload() []byte
	Ack() error
	Nack() error
}

// CommandHandler handles specific command messages received from the broker.
// Implementations of this interface are responsible for the business logic associated with a command type.
type CommandHandler interface {
	Handle(ctx context.Context, msg CommandMessage) error
}

// CommandMessage defines the standard envelope for all internal agent commands.
// It includes metadata for routing, tracing, and sequencing.
type CommandMessage struct {
	CmdType       string          `json:"cmdType"`
	CorrelationID string          `json:"correlationId"`
	Payload       json.RawMessage `json:"payload"`
	Timestamp     int64           `json:"timestamp"`
}
