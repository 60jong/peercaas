package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"agents/internal/core"
	"agents/internal/metrics"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// HeartbeatPayload represents unified telemetry sent to the hub via RabbitMQ.
type HeartbeatPayload struct {
	WorkerID          string              `json:"workerId"`
	AvailableCPU      float64             `json:"availableCpu"`
	AvailableMemoryMb int64               `json:"availableMemoryMb"`
	AverageLatencyMs  float64             `json:"averageLatencyMs"`
	Containers        []ContainerSnapshot `json:"containers"`
}

// ContainerSnapshot provides a point-in-time snapshot of container traffic.
type ContainerSnapshot struct {
	ContainerID string `json:"containerId"`
	TxBytes     int64  `json:"txBytes"`
	RxBytes     int64  `json:"rxBytes"`
}

// HeartbeatManager orchestrates periodic telemetry reporting and time-series shipping.
// It combines real-time resource monitoring with persistent metric storage and remote shipping.
type HeartbeatManager struct {
	mq        core.Broker
	workerID  string
	queue     string
	traffic   *metrics.TrafficStore
	latency   *metrics.LatencyMeasurer
	collector *metrics.Collector
	repo      *metrics.MetricRepository
	shipper   *metrics.MetricShipper
	dockerCli *client.Client
	store     *ContainerStore
}

// NewHeartbeatManager initializes a HeartbeatManager with all necessary backend services.
func NewHeartbeatManager(
	mq core.Broker,
	id, queue string,
	t *metrics.TrafficStore,
	l *metrics.LatencyMeasurer,
	c *metrics.Collector,
	repo *metrics.MetricRepository,
	shipper *metrics.MetricShipper,
	dockerCli *client.Client,
	store *ContainerStore,
) *HeartbeatManager {
	return &HeartbeatManager{
		mq:        mq,
		workerID:  id,
		queue:     queue,
		traffic:   t,
		latency:   l,
		collector: c,
		repo:      repo,
		shipper:   shipper,
		dockerCli: dockerCli,
		store:     store,
	}
}

// Start initiates the periodic reporting and shipping loops. It blocks until the context is cancelled.
func (h *HeartbeatManager) Start(ctx context.Context) {
	reportTicker := time.NewTicker(5 * time.Second)
	defer reportTicker.Stop()

	shipTicker := time.NewTicker(1 * time.Minute)
	defer shipTicker.Stop()

	log.Printf("[Heartbeat] Service started for worker: %s", h.workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[Heartbeat] Service stopping for worker: %s", h.workerID)
			return
		case <-reportTicker.C:
			h.report(ctx)
		case <-shipTicker.C:
			h.ship(ctx)
		}
	}
}

// report collects metrics from all sources, persists them locally, and broadcasts a summary heartbeat.
func (h *HeartbeatManager) report(ctx context.Context) {
	res, err := h.collector.GetAvailableResources(ctx)
	if err != nil {
		log.Printf("[Heartbeat] Resource collection failed: %v", err)
	}

	now := time.Now().UnixNano()
	containers, err := h.dockerCli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		log.Printf("[Heartbeat] Failed to list containers: %v", err)
		return
	}

	snapshots := make([]ContainerSnapshot, 0, len(containers))
	for _, c := range containers {
		stats, err := h.collector.GetContainerStats(ctx, c.ID)
		if err != nil {
			continue
		}

		tx, rx := int64(0), int64(0)
		if t, ok := h.traffic.Get(c.ID); ok {
			tx, rx = t.Tx(), t.Rx()
		}

		clientKey := ""
		if info, ok := h.store.Get(c.ID); ok {
			clientKey = info.ClientKey
		}

		metric := metrics.ContainerMetric{
			WorkerID:    h.workerID,
			ContainerID: c.ID,
			ClientKey:   clientKey,
			CPUUsage:    stats.CPUUsage,
			MemUsageMb:  stats.MemUsageMb,
			NetTxBytes:  tx,
			NetRxBytes:  rx,
			Timestamp:   now,
		}

		if err := h.repo.Save(ctx, metric); err != nil {
			log.Printf("[Heartbeat] Local persistence failed for %s: %v", c.ID[:8], err)
		}

		snapshots = append(snapshots, ContainerSnapshot{
			ContainerID: c.ID,
			TxBytes:     tx,
			RxBytes:     rx,
		})
	}

	payload := HeartbeatPayload{
		WorkerID:          h.workerID,
		AvailableCPU:      res.AvailableCPU,
		AvailableMemoryMb: res.AvailableMemoryMb,
		AverageLatencyMs:  h.latency.Get(),
		Containers:        snapshots,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Heartbeat] Marshal error: %v", err)
		return
	}

	if err := h.mq.Publish(ctx, h.queue, data); err != nil {
		log.Printf("[Heartbeat] RMQ broadcast failed: %v", err)
	}
}

// ship extracts unsent metrics from the local database and forwards them to the remote time-series store.
func (h *HeartbeatManager) ship(ctx context.Context) {
	const batchSize = 100
	totalShipped := 0

	for {
		pending, err := h.repo.GetPending(ctx, batchSize)
		if err != nil {
			log.Printf("[Shipper] Query failed: %v", err)
			break
		}
		if len(pending) == 0 {
			break
		}

		start := time.Now()
		if err := h.shipper.ShipBatch(ctx, pending); err != nil {
			log.Printf("[Shipper] Batch shipping failed: %v", err)
			break
		}

		// Update latency measurements based on shipping response time
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0
		h.latency.Update(elapsed)

		ids := make([]int64, len(pending))
		for i, m := range pending {
			ids[i] = m.ID
		}

		if err := h.repo.MarkSent(ctx, ids); err != nil {
			log.Printf("[Shipper] Mark sent failed: %v", err)
		}
		totalShipped += len(pending)
	}

	if totalShipped > 0 {
		log.Printf("[Shipper] Forwarded %d metrics to VictoriaMetrics", totalShipped)
	}
}
