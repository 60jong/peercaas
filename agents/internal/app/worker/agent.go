package worker

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"agents/internal/config"
	"agents/internal/core"
	"agents/internal/metrics"

	"github.com/docker/docker/client"
)

// WorkerAgent coordinates message handling and telemetry reporting.
type WorkerAgent struct {
	config    config.WorkerConfig
	mq        core.Broker
	dockerCli *client.Client
	handlers  map[string]core.CommandHandler
	heartbeat *HeartbeatManager
	wg        sync.WaitGroup
}

// NewAgent creates a new Worker instance with required dependencies.
func NewAgent(mq core.Broker, cfg config.WorkerConfig, heartbeatQueue string, dockerCli *client.Client, traffic *metrics.TrafficStore, latency *metrics.LatencyMeasurer, repo *metrics.MetricRepository, shipper *metrics.MetricShipper, store *ContainerStore) *WorkerAgent {
	collector := metrics.NewCollector(cfg.MaxCPU, cfg.MaxMemoryMb, dockerCli, store)
	h := NewHeartbeatManager(mq, cfg.WorkerID, heartbeatQueue, traffic, latency, collector, repo, shipper, dockerCli, store)

	return &WorkerAgent{
		config:    cfg,
		mq:        mq,
		dockerCli: dockerCli,
		handlers:  make(map[string]core.CommandHandler),
		heartbeat: h,
	}
}

// Register adds a new command handler to the agent.
func (w *WorkerAgent) Register(cmdType string, handler core.CommandHandler) {
	w.handlers[cmdType] = handler
}

// Run starts the agent's main processing loop.
func (w *WorkerAgent) Run(ctx context.Context, queueName string, traffic *metrics.TrafficStore, latency *metrics.LatencyMeasurer) error {
	// Start the periodic telemetry reporting service
	go w.heartbeat.Start(ctx)

	events, err := w.mq.Subscribe(ctx, queueName)
	if err != nil {
		return err
	}

	log.Printf("[Agent] Listening for commands on: %s", queueName)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Agent] Shutting down...")
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
			log.Printf("[Agent] Panic recovery during event processing: %v", r)
			e.Nack()
		}
	}()

	var msg core.CommandMessage
	if err := json.Unmarshal(e.Payload(), &msg); err != nil {
		log.Printf("[Agent] Invalid command payload: %v", err)
		e.Ack()
		return
	}

	handler, exists := w.handlers[msg.CmdType]
	if !exists {
		log.Printf("[Agent] Unsupported command: %s", msg.CmdType)
		e.Ack()
		return
	}

	log.Printf("[Agent] Executing %s (Trace: %s)", msg.CmdType, msg.TraceID)
	if err := handler.Handle(ctx, msg); err != nil {
		log.Printf("[Agent] Handler error (%s): %v", msg.CmdType, err)
		e.Nack()
	} else {
		e.Ack()
	}
}
