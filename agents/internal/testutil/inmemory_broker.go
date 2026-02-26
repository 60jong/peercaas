// Package testutil은 외부 인프라(RabbitMQ, Docker 등) 없이 테스트할 수 있는
// 인-메모리 구현체들을 제공합니다.
package testutil

import (
	"context"
	"sync"

	"agents/internal/core"
)

// inMemoryEvent는 core.Event의 인-메모리 구현체입니다.
type inMemoryEvent struct {
	payload []byte
}

func (e *inMemoryEvent) Payload() []byte { return e.payload }
func (e *inMemoryEvent) Ack() error      { return nil }
func (e *inMemoryEvent) Nack() error     { return nil }

// InMemoryBroker는 core.Broker의 인-메모리 구현체입니다.
// 테스트에서 RabbitMQ 없이 Worker와 FakeHub 간 메시지 전달에 사용합니다.
type InMemoryBroker struct {
	mu   sync.RWMutex
	subs map[string][]chan core.Event
}

func NewInMemoryBroker() *InMemoryBroker {
	return &InMemoryBroker{
		subs: make(map[string][]chan core.Event),
	}
}

func (b *InMemoryBroker) Connect() error { return nil }
func (b *InMemoryBroker) Close() error   { return nil }

// Publish는 해당 topic을 구독 중인 모든 채널에 메시지를 전달합니다.
func (b *InMemoryBroker) Publish(ctx context.Context, topic string, msg []byte) error {
	b.mu.RLock()
	channels := make([]chan core.Event, len(b.subs[topic]))
	copy(channels, b.subs[topic])
	b.mu.RUnlock()

	evt := &inMemoryEvent{payload: msg}
	for _, ch := range channels {
		select {
		case ch <- evt:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// Subscribe는 topic에 대한 구독 채널을 반환합니다.
// context가 취소되면 채널을 닫고 구독을 해제합니다.
func (b *InMemoryBroker) Subscribe(ctx context.Context, topic string) (<-chan core.Event, error) {
	ch := make(chan core.Event, 16)

	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], ch)
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		defer b.mu.Unlock()
		channels := b.subs[topic]
		for i, c := range channels {
			if c == ch {
				b.subs[topic] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	return ch, nil
}
