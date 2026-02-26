package worker

import (
	"context"
	"encoding/json"
	"log"
	"runtime"
	"time"

	"agents/internal/core"
	"agents/internal/metrics"
)

type HeartbeatPayload struct {
	WorkerID          string              `json:"workerId"`
	AvailableCpu      float64             `json:"availableCpu"`
	AvailableMemoryMb int64               `json:"availableMemoryMb"`
	AverageLatencyMs  float64             `json:"averageLatencyMs"`
	Containers        []ContainerSnapshot `json:"containers"`
}

type ContainerSnapshot struct {
	ContainerID string `json:"containerId"`
	TxBytes     int64  `json:"txBytes"`
	RxBytes     int64  `json:"rxBytes"`
}

func StartHeartbeat(ctx context.Context, mq core.Broker, workerID string, queueName string, traffic *metrics.TrafficStore) {
	ticker := time.NewTicker(5 * time.Second) // 통합 보고이므로 주기를 5초로 단축
	defer ticker.Stop()

	log.Printf("[Worker] Unified Heartbeat started for worker: %s", workerID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendHeartbeat(ctx, mq, workerID, queueName, traffic)
		}
	}
}

func sendHeartbeat(ctx context.Context, mq core.Broker, workerID string, queueName string, traffic *metrics.TrafficStore) {
	// 1. 시스템 리소스 수집
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	availableMem := int64(m.Sys-m.Alloc) / 1024 / 1024

	// 2. 컨테이너 메트릭 수집
	allMetrics := traffic.All()
	containers := make([]ContainerSnapshot, 0, len(allMetrics))
	for _, t := range allMetrics {
		containers = append(containers, ContainerSnapshot{
			ContainerID: t.ContainerID,
			TxBytes:     t.Tx(),
			RxBytes:     t.Rx(),
		})
	}

	payload := HeartbeatPayload{
		WorkerID:          workerID,
		AvailableCpu:      float64(runtime.NumCPU()),
		AvailableMemoryMb: availableMem,
		AverageLatencyMs:  10.5, // TODO: 실측 로직
		Containers:        containers,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Worker] Heartbeat marshal error: %v", err)
		return
	}

	if err := mq.Publish(ctx, queueName, data); err != nil {
		log.Printf("[Worker] Heartbeat publish error: %v", err)
	}
}
