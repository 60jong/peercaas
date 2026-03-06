package metrics

import (
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// ContainerTraffic tracks bidirectional byte counts, connection attempts, and activity timestamps for a specific container.
// It is thread-safe and uses atomic operations for counters to ensure high performance under load.
type ContainerTraffic struct {
	ContainerID string
	StartTime   time.Time

	tx         atomic.Int64
	rx         atomic.Int64
	connCount  atomic.Int32
	lastActive atomic.Int64 // UnixNano

	transportMu sync.RWMutex
	transport   string
}

// newContainerTraffic initializes a new traffic record.
func newContainerTraffic(containerID, transport string) *ContainerTraffic {
	t := &ContainerTraffic{
		ContainerID: containerID,
		StartTime:   time.Now(),
		transport:   transport,
	}
	t.lastActive.Store(time.Now().UnixNano())
	return t
}

// AddTx records outgoing bytes.
func (t *ContainerTraffic) AddTx(n int64) {
	t.tx.Add(n)
	t.lastActive.Store(time.Now().UnixNano())
}

// AddRx records incoming bytes.
func (t *ContainerTraffic) AddRx(n int64) {
	t.rx.Add(n)
	t.lastActive.Store(time.Now().UnixNano())
}

// IncrConn increments the connection counter.
func (t *ContainerTraffic) IncrConn() { t.connCount.Add(1) }

// Tx returns the total transmitted bytes.
func (t *ContainerTraffic) Tx() int64 { return t.tx.Load() }

// Rx returns the total received bytes.
func (t *ContainerTraffic) Rx() int64 { return t.rx.Load() }

// ConnCount returns the total number of connections handled.
func (t *ContainerTraffic) ConnCount() int32 { return t.connCount.Load() }

// LastActive returns the time of the most recent activity.
func (t *ContainerTraffic) LastActive() time.Time {
	ns := t.lastActive.Load()
	if ns == 0 {
		return t.StartTime
	}
	return time.Unix(0, ns)
}

// SetTransport updates the transport type (e.g., "WEBRTC", "RELAY").
func (t *ContainerTraffic) SetTransport(transport string) {
	t.transportMu.Lock()
	defer t.transportMu.Unlock()
	t.transport = transport
}

// Transport returns the current transport type.
func (t *ContainerTraffic) Transport() string {
	t.transportMu.RLock()
	defer t.transportMu.RUnlock()
	return t.transport
}

// TrafficStore maintains a thread-safe registry of traffic statistics for all active containers.
type TrafficStore struct {
	mu   sync.RWMutex
	data map[string]*ContainerTraffic
}

// NewTrafficStore creates a new traffic registry.
func NewTrafficStore() *TrafficStore {
	return &TrafficStore{data: make(map[string]*ContainerTraffic)}
}

// GetOrCreate retrieves an existing traffic record or creates a new one if not found.
func (s *TrafficStore) GetOrCreate(containerID, transport string) *ContainerTraffic {
	s.mu.RLock()
	t, ok := s.data[containerID]
	s.mu.RUnlock()

	if ok {
		if transport != "" {
			t.SetTransport(transport)
		}
		return t
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Double-check after acquiring write lock
	if t, ok = s.data[containerID]; ok {
		return t
	}
	t = newContainerTraffic(containerID, transport)
	s.data[containerID] = t
	return t
}

// Get retrieves a traffic record by container ID.
func (s *TrafficStore) Get(containerID string) (*ContainerTraffic, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.data[containerID]
	return t, ok
}

// All returns a slice containing all traffic records currently in the store.
func (s *TrafficStore) All() []*ContainerTraffic {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ContainerTraffic, 0, len(s.data))
	for _, t := range s.data {
		result = append(result, t)
	}
	return result
}

// Totals calculates the aggregate transmitted and received bytes across all containers.
func (s *TrafficStore) Totals() (tx, rx int64) {
	for _, t := range s.All() {
		tx += t.Tx()
		rx += t.Rx()
	}
	return
}

// countingWriter intercepts writes to update byte counters.
type countingWriter struct {
	w   io.Writer
	add func(int64)
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	if n > 0 {
		c.add(int64(n))
	}
	return n, err
}

// BridgeWithTraffic establishes a bidirectional relay between two ReadWriteClosers.
// It tracks metrics for the connection and logs throughput statistics upon completion.
func BridgeWithTraffic(remote, local io.ReadWriteCloser, stats *ContainerTraffic) {
	defer func() {
		_ = remote.Close()
		_ = local.Close()
	}()

	cwRemote := &countingWriter{w: remote, add: stats.AddTx}
	cwLocal := &countingWriter{w: local, add: stats.AddRx}

	bridgeStart := time.Now()
	log.Printf("[Traffic] Bridge started for container=%s transport=%s", stats.ContainerID, stats.Transport())

	var wg sync.WaitGroup
	wg.Add(2)

	// Local to Remote (TX)
	go func() {
		defer wg.Done()
		start := time.Now()
		txBefore := stats.Tx()
		_, err := io.Copy(cwRemote, local)
		elapsed := time.Since(start)
		txBytes := stats.Tx() - txBefore
		if err != nil && err != io.EOF {
			log.Printf("[Traffic] TX error for container=%s: %v", stats.ContainerID, err)
		}
		log.Printf("[Traffic] TX finished: %s in %v (%.1f KB/s) container=%s",
			FormatBytes(txBytes), elapsed, float64(txBytes)/1024/elapsed.Seconds(), stats.ContainerID)
	}()

	// Remote to Local (RX)
	go func() {
		defer wg.Done()
		start := time.Now()
		rxBefore := stats.Rx()
		_, err := io.Copy(cwLocal, remote)
		elapsed := time.Since(start)
		rxBytes := stats.Rx() - rxBefore
		if err != nil && err != io.EOF {
			log.Printf("[Traffic] RX error for container=%s: %v", stats.ContainerID, err)
		}
		log.Printf("[Traffic] RX finished: %s in %v (%.1f KB/s) container=%s",
			FormatBytes(rxBytes), elapsed, float64(rxBytes)/1024/elapsed.Seconds(), stats.ContainerID)
	}()

	wg.Wait()
	log.Printf("[Traffic] Bridge closed after %v container=%s", time.Since(bridgeStart), stats.ContainerID)
}

// FormatBytes returns a human-readable string representation of a byte count.
func FormatBytes(b int64) string {
	const (
		unit = 1024
	)
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
