package core

import (
	"context"
	"encoding/json"
)

// Broker defines the interface for message queuing operations.
type Broker interface {
	Connect() error
	Close() error
	Publish(ctx context.Context, queue string, body []byte) error
	Subscribe(ctx context.Context, queue string) (<-chan Event, error)
}

// Event represents a single message from the broker.
type Event interface {
	Payload() []byte
	Ack() error
	Nack() error
}

// CommandHandler handles specific command messages received from the broker.
type CommandHandler interface {
	Handle(ctx context.Context, msg CommandMessage) error
}

// CommandMessage defines the standard envelope for all internal agent commands.
type CommandMessage struct {
	CmdType   string          `json:"cmdType"`
	TraceID   string          `json:"traceId"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}
