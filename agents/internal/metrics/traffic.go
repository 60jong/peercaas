package metrics

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// ContainerTraffic tracks bytes sent/received for a single container.
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

func newContainerTraffic(containerID, transport string) *ContainerTraffic {
	t := &ContainerTraffic{
		ContainerID: containerID,
		StartTime:   time.Now(),
		transport:   transport,
	}
	t.lastActive.Store(time.Now().UnixNano())
	return t
}

// AddTx increments outgoing (local→remote) byte count.
func (t *ContainerTraffic) AddTx(n int64) {
	t.tx.Add(n)
	t.lastActive.Store(time.Now().UnixNano())
}

// AddRx increments incoming (remote→local) byte count.
func (t *ContainerTraffic) AddRx(n int64) {
	t.rx.Add(n)
	t.lastActive.Store(time.Now().UnixNano())
}

// IncrConn increments the connection counter by 1.
func (t *ContainerTraffic) IncrConn() { t.connCount.Add(1) }

func (t *ContainerTraffic) Tx() int64        { return t.tx.Load() }
func (t *ContainerTraffic) Rx() int64        { return t.rx.Load() }
func (t *ContainerTraffic) ConnCount() int32 { return t.connCount.Load() }

func (t *ContainerTraffic) LastActive() time.Time {
	ns := t.lastActive.Load()
	if ns == 0 {
		return t.StartTime
	}
	return time.Unix(0, ns)
}

func (t *ContainerTraffic) SetTransport(transport string) {
	t.transportMu.Lock()
	t.transport = transport
	t.transportMu.Unlock()
}

func (t *ContainerTraffic) Transport() string {
	t.transportMu.RLock()
	defer t.transportMu.RUnlock()
	return t.transport
}

// TrafficStore is a thread-safe map of containerID → ContainerTraffic.
// Entries are never removed — historical data is preserved.
type TrafficStore struct {
	mu   sync.RWMutex
	data map[string]*ContainerTraffic
}

func NewTrafficStore() *TrafficStore {
	return &TrafficStore{data: make(map[string]*ContainerTraffic)}
}

// GetOrCreate returns the existing entry or creates a new one.
// If transport is non-empty, the transport field is updated.
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
	if t, ok = s.data[containerID]; ok {
		return t
	}
	t = newContainerTraffic(containerID, transport)
	s.data[containerID] = t
	return t
}

// All returns a snapshot of all ContainerTraffic entries.
func (s *TrafficStore) All() []*ContainerTraffic {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ContainerTraffic, 0, len(s.data))
	for _, t := range s.data {
		result = append(result, t)
	}
	return result
}

// Totals returns the aggregate Tx and Rx bytes across all containers.
func (s *TrafficStore) Totals() (tx, rx int64) {
	for _, t := range s.All() {
		tx += t.Tx()
		rx += t.Rx()
	}
	return
}

// countingWriter wraps an io.Writer and invokes add(n) after each Write.
type countingWriter struct {
	w   io.Writer
	add func(int64)
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.add(int64(n))
	return n, err
}

// BridgeWithTraffic performs a bidirectional relay between remote and local,
// counting bytes in both directions.
//
//	TX (local→remote): bytes written to remote   → stats.AddTx
//	RX (remote→local): bytes written to local    → stats.AddRx
//
// Returns after one direction closes; both connections are closed before returning.
func BridgeWithTraffic(remote, local io.ReadWriteCloser, stats *ContainerTraffic) {
	cwRemote := &countingWriter{w: remote, add: stats.AddTx}
	cwLocal := &countingWriter{w: local, add: stats.AddRx}

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(cwRemote, local) // local → remote (TX)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(cwLocal, remote) // remote → local (RX)
		done <- struct{}{}
	}()
	<-done
	remote.Close()
	local.Close()
}

// FormatBytes returns a human-readable byte count string.
func FormatBytes(b int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
