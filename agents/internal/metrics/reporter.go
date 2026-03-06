package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// Reporter periodic reports traffic metrics to the central Hub.
type Reporter struct {
	hubURL     string
	agentType  string
	agentID    string
	store      *TrafficStore
	httpClient *http.Client
	interval   time.Duration
}

// NewReporter initializes a new metrics reporter.
func NewReporter(hubURL, agentType, agentID string, store *TrafficStore) *Reporter {
	return &Reporter{
		hubURL:    hubURL,
		agentType: agentType,
		agentID:   agentID,
		store:     store,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		interval: 5 * time.Second,
	}
}

// Run starts the reporting loop. It blocks until the context is cancelled.
func (r *Reporter) Run(ctx context.Context) {
	endpoint := fmt.Sprintf("%s/api/v1/metrics", r.hubURL)
	log.Printf("[Metrics] Reporter started → %s (agentType=%s, agentID=%s)", endpoint, r.agentType, r.agentID)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[Metrics] Reporter stopping...")
			return
		case <-ticker.C:
			if err := r.postReport(ctx, endpoint, r.buildReport()); err != nil {
				log.Printf("[Metrics] Report failed: %v", err)
			}
		}
	}
}

func (r *Reporter) buildReport() *metricsReport {
	all := r.store.All()
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
		AgentType:  r.agentType,
		AgentID:    r.agentID,
		Timestamp:  time.Now().Unix(),
		Containers: containers,
	}
}

func (r *Reporter) postReport(ctx context.Context, endpoint string, report *metricsReport) error {
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("hub returned status %d", resp.StatusCode)
	}

	return nil
}

// StartReporter is a convenience function to start a reporter in a background goroutine.
func StartReporter(ctx context.Context, hubURL, agentType, agentID string, store *TrafficStore) {
	reporter := NewReporter(hubURL, agentType, agentID, store)
	go reporter.Run(ctx)
}
