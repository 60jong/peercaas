package rabbitmq

import (
	"context"
	"time"

	"agents/internal/core"

	amqp "github.com/rabbitmq/amqp091-go"
)

// rmqEvent: core.Event 인터페이스 구현체
type rmqEvent struct {
	d amqp.Delivery
}

func (e *rmqEvent) Payload() []byte { return e.d.Body }
func (e *rmqEvent) Ack() error      { return e.d.Ack(false) }
func (e *rmqEvent) Nack() error     { return e.d.Nack(false, false) }

// RabbitMQBroker: core.Broker 인터페이스 구현체
type RabbitMQBroker struct {
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel
}

func New(url string) *RabbitMQBroker {
	return &RabbitMQBroker{url: url}
}

func (r *RabbitMQBroker) Connect() error {
	var err error
	// 연결 재시도 로직은 생략하고 단순 연결
	r.conn, err = amqp.Dial(r.url)
	if err != nil {
		return err
	}
	r.ch, err = r.conn.Channel()
	return err
}

func (r *RabbitMQBroker) Close() error {
	if r.ch != nil {
		r.ch.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// Publish: Client -> Java Server (Queue로 직접 발송 예시)
func (r *RabbitMQBroker) Publish(ctx context.Context, topic string, msg []byte) error {
	// 컨텍스트에 타임아웃이 없으면 안전을 위해 설정
	_, ok := ctx.Deadline()
	if !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	return r.ch.PublishWithContext(ctx,
		"",    // exchange (default)
		topic, // routing key (queue name)
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msg,
		},
	)
}

// Subscribe: Worker <- Java Server
func (r *RabbitMQBroker) Subscribe(ctx context.Context, topic string) (<-chan core.Event, error) {
	// 큐가 없으면 생성 (Durable=true)
	_, err := r.ch.QueueDeclare(topic, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	msgs, err := r.ch.Consume(
		topic,
		"",    // consumer name
		false, // auto-ack (수동 Ack 사용)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, err
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
