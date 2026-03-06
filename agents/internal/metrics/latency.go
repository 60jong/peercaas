package metrics

import (
	"sync"
)

// LatencyMeasurer tracks a moving average (EWMA) of system latency.
// It is thread-safe and provides a smoothed value of recent observations.
type LatencyMeasurer struct {
	mu      sync.RWMutex
	latency float64 // Current EWMA value in milliseconds
	alpha   float64 // Smoothing factor (0 < alpha < 1)
}

// NewLatencyMeasurer creates a new measurer with a default smoothing factor of 0.1.
func NewLatencyMeasurer() *LatencyMeasurer {
	return &LatencyMeasurer{
		alpha: 0.1,
	}
}

// Update incorporates a new latency observation into the moving average.
func (lm *LatencyMeasurer) Update(elapsedMs float64) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.latency == 0.0 {
		// Initialize with the first measurement
		lm.latency = elapsedMs
	} else {
		// Exponential Weighted Moving Average (EWMA)
		lm.latency = lm.alpha*elapsedMs + (1-lm.alpha)*lm.latency
	}
}

// Get returns the current exponentially weighted moving average of latency in milliseconds.
func (lm *LatencyMeasurer) Get() float64 {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.latency
}
