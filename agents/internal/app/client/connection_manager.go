package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v3"
)

// retryIntervals: 30s → 1m → 2m → 2m → ...
var retryIntervals = []time.Duration{
	30 * time.Second,
	1 * time.Minute,
	2 * time.Minute,
}

func retryInterval(attempt int) time.Duration {
	if attempt < len(retryIntervals) {
		return retryIntervals[attempt]
	}
	return retryIntervals[len(retryIntervals)-1]
}

// transport 타입
const (
	transportWebRTC = 0
	transportRelay  = 1
)

// ConnectionManager: WebRTC/Relay 전환 + 백그라운드 WebRTC retry 전략
type ConnectionManager struct {
	config       *Config
	hubClient    HubClientPort
	webRTCConfig *webrtc.Configuration // nil이면 기본값(STUN) 사용

	// 현재 활성 transport (atomic: 0=WebRTC, 1=Relay)
	activeTransport atomic.Int32

	// WebRTC PeerConnection (WebRTC transport 활성 시)
	pcMu sync.RWMutex
	pc   *webrtc.PeerConnection

	// portBindings: "3306/tcp" → containerPort (80, 3306, ...)
	portBindings map[string]int
}

func NewConnectionManager(cfg *Config, hub HubClientPort) *ConnectionManager {
	return &ConnectionManager{
		config:    cfg,
		hubClient: hub,
	}
}

// Run: 메인 진입점
func (cm *ConnectionManager) Run(ctx context.Context) error {
	// 1. 컨테이너 정보 조회
	info, err := cm.hubClient.GetContainerInfo(ctx, cm.config.ContainerID)
	if err != nil {
		return fmt.Errorf("failed to get container info: %w", err)
	}
	log.Printf("[Manager] Container %s ready, ports: %v", info.ContainerID, info.PortBindings)
	cm.portBindings = info.PortBindings

	// 2. TCP 리스너 시작 (transport 전환과 무관하게 동일 포트 유지)
	listeners, err := cm.startListeners(ctx)
	if err != nil {
		return fmt.Errorf("failed to start listeners: %w", err)
	}
	defer func() {
		for _, ln := range listeners {
			ln.Close()
		}
	}()

	// 3. WebRTC 연결 시도
	pc, err := cm.tryWebRTC(ctx)
	if err != nil {
		log.Printf("[Manager] WebRTC failed (%v), switching to relay", err)
		cm.activeTransport.Store(transportRelay)

		// 백그라운드에서 WebRTC retry 시작
		go cm.retryWebRTCLoop(ctx)
	} else {
		log.Printf("[Manager] WebRTC connected")
		cm.setPeerConnection(pc)
		cm.activeTransport.Store(transportWebRTC)

		// WebRTC 연결이 끊기면 relay로 전환
		go cm.watchWebRTCAndFallback(ctx, pc)
	}

	<-ctx.Done()
	return nil
}

// startListeners: 각 포트에 TCP 리스너 시작
func (cm *ConnectionManager) startListeners(ctx context.Context) ([]net.Listener, error) {
	var listeners []net.Listener

	for portKey := range cm.portBindings {
		containerPort, err := parseContainerPort(portKey)
		if err != nil {
			log.Printf("[Manager] Skipping invalid port key %q: %v", portKey, err)
			continue
		}

		addr := fmt.Sprintf("0.0.0.0:%d", containerPort)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("[Manager] Failed to listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, ln)
		log.Printf("[Manager] Listening on %s (portKey: %s)", addr, portKey)

		go cm.acceptLoop(ctx, ln, portKey)
	}

	if len(listeners) == 0 {
		return nil, fmt.Errorf("no TCP listeners started")
	}
	return listeners, nil
}

// acceptLoop: 포트별 연결 수락 루프
func (cm *ConnectionManager) acceptLoop(ctx context.Context, ln net.Listener, portKey string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[Manager] Accept error on %s: %v", portKey, err)
			return
		}
		go cm.handleConnection(ctx, conn, portKey)
	}
}

// handleConnection: 현재 transport에 따라 WebRTC 또는 Relay로 처리
func (cm *ConnectionManager) handleConnection(ctx context.Context, conn net.Conn, portKey string) {
	if cm.activeTransport.Load() == transportWebRTC {
		cm.handleWebRTC(ctx, conn, portKey)
	} else {
		cm.handleRelay(ctx, conn, portKey)
	}
}

// handleWebRTC: WebRTC DataChannel로 릴레이
func (cm *ConnectionManager) handleWebRTC(ctx context.Context, tcpConn net.Conn, portKey string) {
	cm.pcMu.RLock()
	pc := cm.pc
	cm.pcMu.RUnlock()

	if pc == nil || pc.ConnectionState() != webrtc.PeerConnectionStateConnected {
		log.Printf("[Manager] WebRTC not ready, falling back to relay for this connection")
		cm.handleRelay(ctx, tcpConn, portKey)
		return
	}

	dc, err := pc.CreateDataChannel(portKey, nil)
	if err != nil {
		log.Printf("[Manager] DataChannel create failed: %v", err)
		tcpConn.Close()
		return
	}

	dc.OnOpen(func() {
		rawDC, err := dc.Detach()
		if err != nil {
			log.Printf("[Manager] Detach failed: %v", err)
			tcpConn.Close()
			return
		}
		log.Printf("[Manager] WebRTC relay: %s <-> DataChannel(%s)", tcpConn.RemoteAddr(), portKey)
		bridge(rawDC, tcpConn)
	})
}

// handleRelay: Engine TCP relay를 통해 릴레이
func (cm *ConnectionManager) handleRelay(ctx context.Context, tcpConn net.Conn, portKey string) {
	log.Printf("[Manager] Requesting relay session for portKey=%s", portKey)

	relayInfo, err := cm.hubClient.RequestRelay(ctx, cm.config.ContainerID, portKey)
	if err != nil {
		log.Printf("[Manager] Relay request failed: %v", err)
		tcpConn.Close()
		return
	}

	relayAddr := fmt.Sprintf("%s:%d", relayInfo.RelayHost, relayInfo.RelayPort)
	relayConn, err := net.DialTimeout("tcp", relayAddr, 10*time.Second)
	if err != nil {
		log.Printf("[Manager] Failed to connect to relay %s: %v", relayAddr, err)
		tcpConn.Close()
		return
	}

	// 핸드셰이크: 세션 토큰 전송
	if _, err := fmt.Fprintf(relayConn, "%s\n", relayInfo.Token); err != nil {
		log.Printf("[Manager] Relay handshake failed: %v", err)
		relayConn.Close()
		tcpConn.Close()
		return
	}

	log.Printf("[Manager] Relay bridge: %s ↔ relay(token=%s)", tcpConn.RemoteAddr(), relayInfo.Token)
	bridge(relayConn, tcpConn)
}

// tryWebRTC: WebRTC PeerConnection 수립 시도
// 성공 시 PeerConnection 반환, 실패 시 에러
func (cm *ConnectionManager) tryWebRTC(ctx context.Context) (*webrtc.PeerConnection, error) {
	tunnel := &Tunnel{Config: cm.config, HubClient: cm.hubClient, WebRTCConfig: cm.webRTCConfig}
	return tunnel.Connect(ctx)
}

// watchWebRTCAndFallback: WebRTC 연결이 끊기면 relay로 전환 후 retry 루프 시작
func (cm *ConnectionManager) watchWebRTCAndFallback(ctx context.Context, pc *webrtc.PeerConnection) {
	failed := make(chan struct{}, 1)

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateDisconnected ||
			state == webrtc.PeerConnectionStateClosed {
			select {
			case failed <- struct{}{}:
			default:
			}
		}
	})

	select {
	case <-ctx.Done():
		return
	case <-failed:
		log.Printf("[Manager] WebRTC connection lost, switching to relay")
		cm.activeTransport.Store(transportRelay)
		pc.Close()
		go cm.retryWebRTCLoop(ctx)
	}
}

// retryWebRTCLoop: 백그라운드에서 WebRTC 재연결 시도
// 성공하면 hot-swap (새 연결부터 WebRTC 사용)
func (cm *ConnectionManager) retryWebRTCLoop(ctx context.Context) {
	for attempt := 0; ; attempt++ {
		interval := retryInterval(attempt)
		log.Printf("[Manager] WebRTC retry in %v (attempt %d)...", interval, attempt+1)

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}

		pc, err := cm.tryWebRTC(ctx)
		if err != nil {
			log.Printf("[Manager] WebRTC retry %d failed: %v", attempt+1, err)
			continue
		}

		// 성공 — hot-swap: 새 연결부터 WebRTC 사용
		log.Printf("[Manager] WebRTC reconnected! Hot-swapping to WebRTC transport")
		cm.setPeerConnection(pc)
		cm.activeTransport.Store(transportWebRTC)

		// 새 WebRTC도 끊기면 다시 relay로
		go cm.watchWebRTCAndFallback(ctx, pc)
		return
	}
}

func (cm *ConnectionManager) setPeerConnection(pc *webrtc.PeerConnection) {
	cm.pcMu.Lock()
	defer cm.pcMu.Unlock()
	if cm.pc != nil {
		cm.pc.Close()
	}
	cm.pc = pc
}

// bridge: 두 io.ReadWriteCloser 간 양방향 복사
func bridge(a, b io.ReadWriteCloser) {
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(a, b)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(b, a)
		done <- struct{}{}
	}()
	<-done
	a.Close()
	b.Close()
}
