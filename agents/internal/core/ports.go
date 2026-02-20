package core

import "context"

// Event: MQ 수신 메시지 추상화
type Event interface {
	Payload() []byte
	Ack() error
	Nack() error
}

// Broker: MQ 인프라 추상화 (Worker는 Subscribe, Client는 Publish 사용)
type Broker interface {
	Connect() error
	Close() error
	Publish(ctx context.Context, topic string, msg []byte) error
	Subscribe(ctx context.Context, topic string) (<-chan Event, error)
}

// CommandHandler: Worker가 처리할 비즈니스 로직 인터페이스
type CommandHandler interface {
	Handle(ctx context.Context, msg CommandMessage) error
}
