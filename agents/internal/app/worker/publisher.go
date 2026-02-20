package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"agents/internal/core"
)

// BrokerResultPublisher: core.Broker를 이용한 ResultPublisher 구현체
type BrokerResultPublisher struct {
	Broker    core.Broker
	QueueName string
}

func (p *BrokerResultPublisher) PublishResult(msg core.CommandMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal command message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return p.Broker.Publish(ctx, p.QueueName, data)
}
