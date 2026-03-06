package worker

import (
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
)

// ContainerInfo stores metadata about a running container managed by this worker.
type ContainerInfo struct {
	ContainerID   string
	CorrelationID string
	ClientKey     string
	ContainerPort int
	PublicPort    int
	Name          string
	PortBindings  map[string]int

	mu              sync.Mutex
	peerConnections []*webrtc.PeerConnection
}

// ContainerStore provides a thread-safe registry for mapping container IDs to their metadata and active connections.
type ContainerStore struct {
	mu   sync.RWMutex
	data map[string]*ContainerInfo
}

// NewContainerStore initializes a new empty container registry.
func NewContainerStore() *ContainerStore {
	return &ContainerStore{
		data: make(map[string]*ContainerInfo),
	}
}

// Add inserts a container record into the store.
func (s *ContainerStore) Add(info *ContainerInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[info.ContainerID] = info
}

// Get retrieves a container record by its ID.
func (s *ContainerStore) Get(containerID string) (*ContainerInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, ok := s.data[containerID]
	return info, ok
}

// Delete removes a container record and closes all its active connections.
func (s *ContainerStore) Delete(containerID string) {
	s.mu.Lock()
	_, ok := s.data[containerID]
	delete(s.data, containerID)
	s.mu.Unlock()

	if ok {
		s.ClosePeerConnections(containerID)
	}
}

// AddPeerConnection associates a new WebRTC PeerConnection with a specific container.
func (s *ContainerStore) AddPeerConnection(containerID string, pc *webrtc.PeerConnection) {
	s.mu.RLock()
	info, ok := s.data[containerID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	info.mu.Lock()
	defer info.mu.Unlock()
	info.peerConnections = append(info.peerConnections, pc)
}

// ClosePeerConnections terminates all WebRTC connections associated with a specific container.
func (s *ContainerStore) ClosePeerConnections(containerID string) {
	s.mu.RLock()
	info, ok := s.data[containerID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	info.mu.Lock()
	pcs := info.peerConnections
	info.peerConnections = nil
	info.mu.Unlock()

	for _, pc := range pcs {
		if err := pc.Close(); err != nil {
			log.Printf("[Store] Error closing PeerConnection for %s: %v", containerID[:12], err)
		}
	}
}

// All returns a snapshot of all container records currently in the store.
func (s *ContainerStore) All() []*ContainerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ContainerInfo, 0, len(s.data))
	for _, info := range s.data {
		result = append(result, info)
	}
	return result
}
