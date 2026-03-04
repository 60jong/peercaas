package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MetricShipper sends local metrics to central VictoriaMetrics.
type MetricShipper struct {
	vmEndpoint string
	user       string
	password   string
	httpClient *http.Client
}

// NewMetricShipper creates a new shipper instance with optional Basic Auth.
func NewMetricShipper(endpoint, user, password string) *MetricShipper {
	return &MetricShipper{
		vmEndpoint: endpoint,
		user:       user,
		password:   password,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ShipBatch converts metrics to Influx Line Protocol and sends to VM.
func (s *MetricShipper) ShipBatch(ctx context.Context, metrics []ContainerMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, m := range metrics {
		// Influx Line Protocol: measurement,tags fields timestamp
		line := fmt.Sprintf("container_usage,worker_id=%s,container_id=%s,client_key=%s cpu_usage=%.2f,mem_usage_mb=%d,net_tx_bytes=%d,net_rx_bytes=%d %d\n",
			m.WorkerID, m.ContainerID, m.ClientKey, m.CPUUsage, m.MemUsageMb, m.NetTxBytes, m.NetRxBytes, m.Timestamp)
		buf.WriteString(line)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.vmEndpoint, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	// Apply Basic Auth if credentials provided
	if s.user != "" && s.password != "" {
		req.SetBasicAuth(s.user, s.password)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("victoria-metrics error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
