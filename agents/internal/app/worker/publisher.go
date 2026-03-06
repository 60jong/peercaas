package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"agents/internal/core"
)

// ResultPublisher defines the interface for sending command results back to the Hub.
type ResultPublisher interface {
	Publish(ctx context.Context, correlationID string, cmdType string, payload any) error
}

// BrokerResultPublisher implements ResultPublisher using a message broker.
type BrokerResultPublisher struct {
	Broker    core.Broker
	QueueName string
}

// Publish wraps a result in a CommandMessage and sends it to the configured result queue.
func (p *BrokerResultPublisher) Publish(ctx context.Context, correlationID string, cmdType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal result payload: %w", err)
	}

	msg := core.CommandMessage{
		CmdType:       cmdType,
		CorrelationID: correlationID,
		Payload:       data,
		Timestamp:     0, // Will be set by core or broker if needed
	}

	envelope, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal result message: %w", err)
	}

	return p.Broker.Publish(ctx, p.QueueName, envelope)
}
