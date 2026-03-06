// Package worker implements the worker agent that executes container operations and reports telemetry.
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

// WorkerAgent coordinates message handling and telemetry reporting for a single worker node.
// It manages command routing, periodic heartbeats, and resource monitoring.
type WorkerAgent struct {
	config    config.WorkerConfig
	mq        core.Broker
	dockerCli *client.Client
	handlers  map[string]core.CommandHandler
	heartbeat *HeartbeatManager
	wg        sync.WaitGroup
}

// NewAgent initializes a new WorkerAgent with its required dependencies.
func NewAgent(
	mq core.Broker,
	cfg config.WorkerConfig,
	heartbeatQueue string,
	dockerCli *client.Client,
	traffic *metrics.TrafficStore,
	latency *metrics.LatencyMeasurer,
	repo *metrics.MetricRepository,
	shipper *metrics.MetricShipper,
	store *ContainerStore,
) *WorkerAgent {
	collector := metrics.NewCollector(cfg.MaxCPU, cfg.MaxMemoryMb, dockerCli)
	h := NewHeartbeatManager(mq, cfg.WorkerID, heartbeatQueue, traffic, latency, collector, repo, shipper, dockerCli, store)

	return &WorkerAgent{
		config:    cfg,
		mq:        mq,
		dockerCli: dockerCli,
		handlers:  make(map[string]core.CommandHandler),
		heartbeat: h,
	}
}

// Register maps a command type to a specific handler.
func (w *WorkerAgent) Register(cmdType string, handler core.CommandHandler) {
	w.handlers[cmdType] = handler
}

// Run starts the agent's main processing loop. It blocks until the context is cancelled.
// It listens for commands on the specified queue and processes them concurrently.
func (w *WorkerAgent) Run(ctx context.Context, queueName string, traffic *metrics.TrafficStore, latency *metrics.LatencyMeasurer) error {
	// Start the periodic telemetry and heartbeat service
	go w.heartbeat.Start(ctx)

	events, err := w.mq.Subscribe(ctx, queueName)
	if err != nil {
		return err
	}

	log.Printf("[Agent] Listening for commands on: %s", queueName)

	// Simple semaphore for concurrency control
	concurrency := w.config.Concurrency
	if concurrency <= 0 {
		concurrency = 10 // Default
	}
	sem := make(chan struct{}, concurrency)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Agent] Shutdown initiated, waiting for active tasks...")
			w.wg.Wait()
			return nil

		case evt, ok := <-events:
			if !ok {
				return nil
			}

			w.wg.Add(1)
			sem <- struct{}{} // Acquire token

			go func(e core.Event) {
				defer w.wg.Done()
				defer func() { <-sem }() // Release token
				w.processEvent(ctx, e)
			}(evt)
		}
	}
}

func (w *WorkerAgent) processEvent(ctx context.Context, e core.Event) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Agent] Panic recovery during event processing: %v", r)
			_ = e.Nack()
		}
	}()

	var msg core.CommandMessage
	if err := json.Unmarshal(e.Payload(), &msg); err != nil {
		log.Printf("[Agent] Invalid command payload: %v", err)
		_ = e.Ack()
		return
	}

	handler, exists := w.handlers[msg.CmdType]
	if !exists {
		log.Printf("[Agent] Unsupported command: %s", msg.CmdType)
		_ = e.Ack()
		return
	}

	log.Printf("[Agent] Executing %s (CorrelationID: %s)", msg.CmdType, msg.CorrelationID)
	if err := handler.Handle(ctx, msg); err != nil {
		log.Printf("[Agent] Handler error (%s): %v", msg.CmdType, err)
		_ = e.Nack()
	} else {
		_ = e.Ack()
	}
}
