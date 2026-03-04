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
	AvailableCpu      float64             `json:"availableCpu"`
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

// NewHeartbeatManager creates a new reporting instance with SQLite and VM shipping.
func NewHeartbeatManager(mq core.Broker, id, queue string, t *metrics.TrafficStore, l *metrics.LatencyMeasurer, c *metrics.Collector, repo *metrics.MetricRepository, shipper *metrics.MetricShipper, dockerCli *client.Client, store *ContainerStore) *HeartbeatManager {
	return &HeartbeatManager{mq, id, queue, t, l, c, repo, shipper, dockerCli, store}
}

// Start initiates the reporting and shipping loops.
func (h *HeartbeatManager) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Shipping ticker (every 1 minute)
	shipTicker := time.NewTicker(1 * time.Minute)
	defer shipTicker.Stop()

	log.Printf("[Heartbeat] Reporting service started for worker: %s", h.workerID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.report(ctx)
		case <-shipTicker.C:
			h.ship(ctx)
		}
	}
}

// report collects metrics, saves to SQLite, and sends heartbeat via RMQ.
func (h *HeartbeatManager) report(ctx context.Context) {
	// 1. Gather resource availability
	res, _ := h.collector.GetAvailableResources(ctx)

	now := time.Now().UnixNano()
	
	// 2. Gather ALL running containers from Docker to ensure we record metrics even without traffic
	containers, err := h.dockerCli.ContainerList(ctx, container.ListOptions{})
	snapshotList := make([]ContainerSnapshot, 0)
	
	if err == nil {
		for _, c := range containers {
			// Get individual stats (CPU/Mem)
			cStats, err := h.collector.GetContainerStats(ctx, c.ID)
			if err != nil {
				continue
			}

			// Get network stats if available in traffic store
			tx, rx := int64(0), int64(0)
			if t, ok := h.traffic.Get(c.ID); ok {
				tx = t.Tx()
				rx = t.Rx()
			}

			// Get clientKey from store if available
			clientKey := ""
			if info, ok := h.store.Get(c.ID); ok {
				clientKey = info.ClientKey
			}

			// Save to local SQLite for time-series / billing
			m := metrics.ContainerMetric{
				WorkerID:    h.workerID,
				ContainerID: c.ID,
				ClientKey:   clientKey,
				CPUUsage:    cStats.CPUUsage,
				MemUsageMb:  cStats.MemUsageMb,
				NetTxBytes:  tx,
				NetRxBytes:  rx,
				Timestamp:   now,
			}
			if err := h.repo.Save(ctx, m); err != nil {
				log.Printf("[Heartbeat] Failed to save metric for %s: %v", c.ID[:8], err)
			}

			snapshotList = append(snapshotList, ContainerSnapshot{
				ContainerID: c.ID,
				TxBytes:     tx,
				RxBytes:     rx,
			})
		}
	}

	// 3. Send Summary Heartbeat via RMQ
	payload := HeartbeatPayload{
		WorkerID:          h.workerID,
		AvailableCpu:      res.AvailableCPU,
		AvailableMemoryMb: res.AvailableMemoryMb,
		AverageLatencyMs:  h.latency.Get(),
		Containers:        snapshotList,
	}

	data, _ := json.Marshal(payload)
	
	if err := h.mq.Publish(ctx, h.queue, data); err != nil {
		log.Printf("[Heartbeat] Failed to publish metrics to RMQ: %v", err)
	}
}

// ship reads pending metrics from SQLite and sends to VictoriaMetrics.
func (h *HeartbeatManager) ship(ctx context.Context) {
	totalShipped := 0
	for {
		pending, err := h.repo.GetPending(ctx, 100)
		if err != nil {
			log.Printf("[Shipper] Failed to fetch pending metrics: %v", err)
			break
		}
		if len(pending) == 0 {
			break
		}

		// Measure VictoriaMetrics ship time as the current latency (New Piggybacking Strategy)
		start := time.Now()
		if err := h.shipper.ShipBatch(ctx, pending); err != nil {
			log.Printf("[Shipper] Failed to ship batch: %v", err)
			break
		} else {
			elapsed := float64(time.Since(start).Microseconds()) / 1000.0
			h.latency.Update(elapsed)
		}

		ids := make([]int64, len(pending))
		for i, m := range pending {
			ids[i] = m.ID
		}
		if err := h.repo.MarkSent(ctx, ids); err != nil {
			log.Printf("[Shipper] Failed to mark metrics as sent: %v", err)
		}
		totalShipped += len(pending)
	}
	
	if totalShipped > 0 {
		log.Printf("[Shipper] Successfully shipped %d metrics to VictoriaMetrics", totalShipped)
	}
}
