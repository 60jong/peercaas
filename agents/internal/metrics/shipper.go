package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MetricShipper sends local metrics to central VictoriaMetrics using the Influx Line Protocol.
type MetricShipper struct {
	vmEndpoint string
	user       string
	password   string
	httpClient *http.Client
}

// NewMetricShipper creates a new shipper instance with a pre-configured timeout.
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

// ShipBatch converts a slice of metrics to Influx Line Protocol and sends them to VictoriaMetrics in a single request.
func (s *MetricShipper) ShipBatch(ctx context.Context, metrics []ContainerMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, m := range metrics {
		// Influx Line Protocol format: measurement,tags fields timestamp
		// Example: container_usage,worker_id=w1,container_id=c1 cpu_usage=0.5,mem_usage_mb=128 1625097600000000000
		_, _ = fmt.Fprintf(&buf, "container_usage,worker_id=%s,container_id=%s,client_key=%s cpu_usage=%.2f,mem_usage_mb=%d,net_tx_bytes=%d,net_rx_bytes=%d %d\n",
			m.WorkerID, m.ContainerID, m.ClientKey, m.CPUUsage, m.MemUsageMb, m.NetTxBytes, m.NetRxBytes, m.Timestamp)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.vmEndpoint, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	if s.user != "" && s.password != "" {
		req.SetBasicAuth(s.user, s.password)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1024)) // Limit error body size
		if err != nil {
			return fmt.Errorf("victoria-metrics error (status %d): (failed to read body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("victoria-metrics error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
