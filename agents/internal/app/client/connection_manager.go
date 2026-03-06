package client

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"agents/internal/metrics"

	"github.com/pion/webrtc/v3"
)

type transportType int32

const (
	transportWebRTC transportType = 0
	transportRelay  transportType = 1
)

// ConnectionManager handles the orchestration between WebRTC and fallback Relay transports.
// It manages TCP listeners, performs hot-swapping between transports, and handles reconnections.
type ConnectionManager struct {
	config    *Config
	hubClient *HubClient
	traffic   *metrics.TrafficStore

	activeTransport int32 // atomic transportType
	pcMu            sync.RWMutex
	pc              *webrtc.PeerConnection
	portBindings    map[string]int
}

// NewConnectionManager initializes a new ConnectionManager.
func NewConnectionManager(cfg *Config, hub *HubClient, traffic *metrics.TrafficStore) *ConnectionManager {
	return &ConnectionManager{
		config:    cfg,
		hubClient: hub,
		traffic:   traffic,
	}
}

// Run starts the management loop, initializes listeners, and manages the primary transport lifecycle.
func (cm *ConnectionManager) Run(ctx context.Context) error {
	info, err := cm.hubClient.GetContainerInfo(ctx, cm.config.ContainerID)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	log.Printf("[Manager] Container %s ready, ports: %v", info.ContainerID[:12], info.PortBindings)
	cm.portBindings = info.PortBindings

	listeners, err := cm.startListeners(ctx)
	if err != nil {
		return fmt.Errorf("failed to start listeners: %w", err)
	}
	defer func() {
		for _, ln := range listeners {
			_ = ln.Close()
		}
	}()

	// Attempt initial WebRTC connection
	if pc, err := cm.tryWebRTC(ctx); err != nil {
		log.Printf("[Manager] Initial WebRTC failed: %v. Using relay.", err)
		atomic.StoreInt32(&cm.activeTransport, int32(transportRelay))
		go cm.reconnectLoop(ctx)
	} else {
		log.Printf("[Manager] WebRTC connected")
		cm.setPeerConnection(pc)
		atomic.StoreInt32(&cm.activeTransport, int32(transportWebRTC))
		go cm.monitorWebRTC(ctx, pc)
	}

	<-ctx.Done()
	return nil
}

func (cm *ConnectionManager) startListeners(ctx context.Context) ([]net.Listener, error) {
	var listeners []net.Listener
	for key := range cm.portBindings {
		port, err := parseContainerPort(key)
		if err != nil {
			continue
		}

		addr := fmt.Sprintf("0.0.0.0:%d", port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("[Manager] Listen failed on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, ln)
		log.Printf("[Manager] Listening on %s (%s)", addr, key)

		go cm.acceptLoop(ctx, ln, key)
	}

	if len(listeners) == 0 {
		return nil, fmt.Errorf("no listeners could be established")
	}
	return listeners, nil
}

func (cm *ConnectionManager) acceptLoop(ctx context.Context, ln net.Listener, portKey string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		go cm.handleNewConnection(ctx, conn, portKey)
	}
}

func (cm *ConnectionManager) handleNewConnection(ctx context.Context, conn net.Conn, portKey string) {
	if transportType(atomic.LoadInt32(&cm.activeTransport)) == transportWebRTC {
		cm.handleWebRTC(ctx, conn, portKey)
	} else {
		cm.handleRelay(ctx, conn, portKey)
	}
}

func (cm *ConnectionManager) handleWebRTC(ctx context.Context, tcpConn net.Conn, portKey string) {
	cm.pcMu.RLock()
	pc := cm.pc
	cm.pcMu.RUnlock()

	if pc == nil || pc.ConnectionState() != webrtc.PeerConnectionStateConnected {
		cm.handleRelay(ctx, tcpConn, portKey)
		return
	}

	dc, err := pc.CreateDataChannel(portKey, nil)
	if err != nil {
		_ = tcpConn.Close()
		return
	}

	dc.OnOpen(func() {
		raw, err := dc.Detach()
		if err != nil {
			_ = tcpConn.Close()
			return
		}
		stats := cm.traffic.GetOrCreate(cm.config.ContainerID, "WEBRTC")
		stats.IncrConn()
		metrics.BridgeWithTraffic(raw, tcpConn, stats)
	})
}

func (cm *ConnectionManager) handleRelay(ctx context.Context, tcpConn net.Conn, portKey string) {
	defer tcpConn.Close()

	info, err := cm.hubClient.RequestRelay(ctx, cm.config.ContainerID, portKey)
	if err != nil {
		return
	}

	addr := fmt.Sprintf("%s:%d", info.RelayHost, info.RelayPort)
	relayConn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return
	}

	if _, err := fmt.Fprintf(relayConn, "%s\n", info.Token); err != nil {
		_ = relayConn.Close()
		return
	}

	stats := cm.traffic.GetOrCreate(cm.config.ContainerID, "RELAY")
	stats.IncrConn()
	metrics.BridgeWithTraffic(relayConn, tcpConn, stats)
}

func (cm *ConnectionManager) tryWebRTC(ctx context.Context) (*webrtc.PeerConnection, error) {
	t := &Tunnel{Config: cm.config, HubClient: cm.hubClient}
	return t.Connect(ctx)
}

func (cm *ConnectionManager) monitorWebRTC(ctx context.Context, pc *webrtc.PeerConnection) {
	done := make(chan struct{})
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateDisconnected || state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	select {
	case <-ctx.Done():
	case <-done:
		log.Printf("[Manager] WebRTC connection lost, switching to relay")
		atomic.StoreInt32(&cm.activeTransport, int32(transportRelay))
		_ = pc.Close()
		go cm.reconnectLoop(ctx)
	}
}

func (cm *ConnectionManager) reconnectLoop(ctx context.Context) {
	backoff := []time.Duration{10 * time.Second, 30 * time.Second, 1 * time.Minute, 2 * time.Minute}
	idx := 0

	for {
		timer := time.NewTimer(backoff[idx])
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		if pc, err := cm.tryWebRTC(ctx); err == nil {
			log.Printf("[Manager] WebRTC reconnected, hot-swapping...")
			cm.setPeerConnection(pc)
			atomic.StoreInt32(&cm.activeTransport, int32(transportWebRTC))
			go cm.monitorWebRTC(ctx, pc)
			return
		}

		if idx < len(backoff)-1 {
			idx++
		}
	}
}

func (cm *ConnectionManager) setPeerConnection(pc *webrtc.PeerConnection) {
	cm.pcMu.Lock()
	defer cm.pcMu.Unlock()
	if cm.pc != nil {
		_ = cm.pc.Close()
	}
	cm.pc = pc
}
