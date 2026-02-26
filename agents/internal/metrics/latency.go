package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// LatencyMeasurer measures RTT to the Hub using EWMA smoothing.
type LatencyMeasurer struct {
	hubPingURL string
	alpha      float64 // EWMA smoothing factor (0 < alpha <= 1)
	interval   time.Duration

	mu      sync.RWMutex
	latency float64 // current EWMA latency in ms
	client  *http.Client
}

// NewLatencyMeasurer creates a new LatencyMeasurer.
// hubURL should be the base URL of the hub (e.g., "http://localhost:8080").
func NewLatencyMeasurer(hubURL string) *LatencyMeasurer {
	return &LatencyMeasurer{
		hubPingURL: fmt.Sprintf("%s/api/health/ping", hubURL),
		alpha:      0.3,
		interval:   10 * time.Second,
		latency:    0.0,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Start begins periodic latency measurement in a loop until ctx is cancelled.
func (lm *LatencyMeasurer) Start(ctx context.Context) {
	log.Printf("[Latency] Starting measurement to %s (interval: %v)", lm.hubPingURL, lm.interval)

	// Measure once immediately
	lm.measure()

	ticker := time.NewTicker(lm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Latency] Measurement stopped")
			return
		case <-ticker.C:
			lm.measure()
		}
	}
}

// Get returns the current EWMA latency in milliseconds.
func (lm *LatencyMeasurer) Get() float64 {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.latency
}

func (lm *LatencyMeasurer) measure() {
	start := time.Now()
	resp, err := lm.client.Get(lm.hubPingURL)
	elapsed := float64(time.Since(start).Microseconds()) / 1000.0 // ms with sub-ms precision

	if err != nil {
		log.Printf("[Latency] Ping failed: %v", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Latency] Ping returned status %d", resp.StatusCode)
		return
	}

	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.latency == 0.0 {
		lm.latency = elapsed // first measurement
	} else {
		lm.latency = lm.alpha*elapsed + (1-lm.alpha)*lm.latency // EWMA
	}

	log.Printf("[Latency] RTT=%.2fms, EWMA=%.2fms", elapsed, lm.latency)
}
