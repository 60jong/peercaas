package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type metricsReport struct {
	AgentType  string              `json:"agentType"`
	AgentID    string              `json:"agentId"`
	Timestamp  int64               `json:"timestamp"`
	Containers []containerSnapshot `json:"containers"`
}

type containerSnapshot struct {
	ContainerID string `json:"containerId"`
	Transport   string `json:"transport"`
	TxBytes     int64  `json:"txBytes"`
	RxBytes     int64  `json:"rxBytes"`
	ConnCount   int32  `json:"connCount"`
	StartTime   string `json:"startTime"`  // RFC3339
	LastActive  string `json:"lastActive"` // RFC3339
}

// StartReporter starts a background goroutine that POSTs traffic metrics to Hub
// every 5 seconds. agentType is "client" or "worker"; agentID is containerID or workerID.
func StartReporter(ctx context.Context, hubURL, agentType, agentID string, store *TrafficStore) {
	endpoint := hubURL + "/api/v1/metrics"
	log.Printf("[Metrics] Reporter started → %s (agentType=%s, agentID=%s)", endpoint, agentType, agentID)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				postReport(endpoint, buildReport(agentType, agentID, store))
			}
		}
	}()
}

func buildReport(agentType, agentID string, store *TrafficStore) *metricsReport {
	all := store.All()
	containers := make([]containerSnapshot, 0, len(all))
	for _, t := range all {
		containers = append(containers, containerSnapshot{
			ContainerID: t.ContainerID,
			Transport:   t.Transport(),
			TxBytes:     t.Tx(),
			RxBytes:     t.Rx(),
			ConnCount:   t.ConnCount(),
			StartTime:   t.StartTime.UTC().Format(time.RFC3339),
			LastActive:  t.LastActive().UTC().Format(time.RFC3339),
		})
	}
	return &metricsReport{
		AgentType:  agentType,
		AgentID:    agentID,
		Timestamp:  time.Now().Unix(),
		Containers: containers,
	}
}

func postReport(endpoint string, report *metricsReport) {
	body, err := json.Marshal(report)
	if err != nil {
		log.Printf("[Metrics] Marshal error: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[Metrics] Report failed: %v", err)
		return
	}
	resp.Body.Close()
}
