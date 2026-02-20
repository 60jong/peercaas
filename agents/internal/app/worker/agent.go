package worker

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"agents/internal/core"
)

type WorkerAgent struct {
	mq       core.Broker
	handlers map[string]core.CommandHandler
	wg       sync.WaitGroup
}

func NewAgent(mq core.Broker) *WorkerAgent {
	return &WorkerAgent{
		mq:       mq,
		handlers: make(map[string]core.CommandHandler),
	}
}

func (w *WorkerAgent) Register(cmdType string, handler core.CommandHandler) {
	w.handlers[cmdType] = handler
}

func (w *WorkerAgent) Run(ctx context.Context, queueName string) error {
	events, err := w.mq.Subscribe(ctx, queueName)
	if err != nil {
		return err
	}

	log.Printf("[Worker] Listening on queue: %s", queueName)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Worker] Shutting down...")
			w.wg.Wait()
			return nil

		case evt, ok := <-events:
			if !ok {
				return nil
			}
			w.wg.Add(1)
			go func(e core.Event) {
				defer w.wg.Done()
				w.processEvent(ctx, e)
			}(evt)
		}
	}
}

func (w *WorkerAgent) processEvent(ctx context.Context, e core.Event) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Worker] Panic: %v", r)
			e.Nack() // 재시도 정책에 따라 Nack 또는 Ack 처리
		}
	}()

	var msg core.CommandMessage
	if err := json.Unmarshal(e.Payload(), &msg); err != nil {
		log.Printf("[Worker] JSON Error: %v", err)
		e.Ack() // 형식이 잘못된 메시지는 버림
		return
	}

	handler, exists := w.handlers[msg.CmdType]
	if !exists {
		log.Printf("[Worker] Unknown command: %s", msg.CmdType)
		e.Ack()
		return
	}

	log.Printf("[Worker] Processing %s (Trace: %s)", msg.CmdType, msg.TraceID)
	if err := handler.Handle(ctx, msg); err != nil {
		log.Printf("[Worker] Handler Error: %v", err)
		e.Nack() // 실패 시 재시도
	} else {
		e.Ack() // 성공
	}
}
