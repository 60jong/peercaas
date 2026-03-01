package metrics

import (
	"sync"
)

// LatencyMeasurer tracks a moving average (EWMA) of latency.
type LatencyMeasurer struct {
	mu      sync.RWMutex
	latency float64 // in milliseconds
	alpha   float64 // EWMA alpha (e.g., 0.1)
}

// NewLatencyMeasurer creates a new measurer.
func NewLatencyMeasurer() *LatencyMeasurer {
	return &LatencyMeasurer{
		alpha: 0.1, // Smoothing factor
	}
}

// Update records a new latency measurement and updates the EWMA.
func (lm *LatencyMeasurer) Update(elapsedMs float64) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.latency == 0.0 {
		lm.latency = elapsedMs // first measurement
	} else {
		lm.latency = lm.alpha*elapsedMs + (1-lm.alpha)*lm.latency // EWMA
	}
}

// Get returns the current EWMA latency in milliseconds.
func (lm *LatencyMeasurer) Get() float64 {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.latency
}
