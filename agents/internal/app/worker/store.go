package worker

import (
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
)

type ContainerInfo struct {
	ContainerID  string
	Name         string
	ClientKey    string
	PortBindings map[string]int // "3306/tcp" -> 33060
	TraceID      string
	PeerConns    []*webrtc.PeerConnection
}

type ContainerStore struct {
	mu         sync.RWMutex
	containers map[string]*ContainerInfo // key: containerId
}

func NewContainerStore() *ContainerStore {
	return &ContainerStore{
		containers: make(map[string]*ContainerInfo),
	}
}

func (s *ContainerStore) Put(info *ContainerInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.containers[info.ContainerID] = info
}

func (s *ContainerStore) Get(containerID string) (*ContainerInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, ok := s.containers[containerID]
	return info, ok
}

func (s *ContainerStore) Delete(containerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.containers, containerID)
}

func (s *ContainerStore) AddPeerConnection(containerID string, pc *webrtc.PeerConnection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.containers[containerID]
	if !ok {
		log.Printf("[Store] Container %s not found, cannot add PeerConnection", containerID)
		return
	}
	info.PeerConns = append(info.PeerConns, pc)
}

func (s *ContainerStore) ClosePeerConnections(containerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.containers[containerID]
	if !ok {
		return
	}
	for _, pc := range info.PeerConns {
		if err := pc.Close(); err != nil {
			log.Printf("[Store] Failed to close PeerConnection for %s: %v", containerID, err)
		}
	}
	info.PeerConns = nil
}
