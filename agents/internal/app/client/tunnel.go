package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

type Tunnel struct {
	Config    *Config
	HubClient *HubClient
}

// Start: 단독 실행 (ConnectionManager 없이 WebRTC only 모드)
func (t *Tunnel) Start(ctx context.Context) error {
	info, err := t.HubClient.GetContainerInfo(ctx, t.Config.ContainerID)
	if err != nil {
		return fmt.Errorf("failed to get container info: %w", err)
	}
	log.Printf("[Tunnel] Container %s is RUNNING on worker %s, portBindings: %v",
		info.ContainerID, info.WorkerID, info.PortBindings)

	pc, err := t.Connect(ctx)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateDisconnected {
			cancel()
		}
	})

	var listeners []net.Listener
	var wg sync.WaitGroup

	for key := range info.PortBindings {
		containerPort, err := parseContainerPort(key)
		if err != nil {
			log.Printf("[Tunnel] Skipping invalid port key %q: %v", key, err)
			continue
		}

		addr := fmt.Sprintf("0.0.0.0:%d", containerPort)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("[Tunnel] Failed to listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, ln)
		log.Printf("[Tunnel] Listening on %s for port key %q", addr, key)

		wg.Add(1)
		go func(ln net.Listener, portKey string) {
			defer wg.Done()
			t.acceptLoop(ctx, pc, ln, portKey)
		}(ln, key)
	}

	if len(listeners) == 0 {
		pc.Close()
		return fmt.Errorf("no TCP listeners started")
	}

	<-ctx.Done()
	log.Println("[Tunnel] Shutting down...")
	for _, ln := range listeners {
		ln.Close()
	}
	wg.Wait()
	pc.Close()
	log.Println("[Tunnel] Shutdown complete")
	return nil
}

// Connect: PeerConnection 수립만 담당 (ConnectionManager에서 호출)
// 성공 시 Connected 상태의 PeerConnection 반환
func (t *Tunnel) Connect(ctx context.Context) (*webrtc.PeerConnection, error) {
	se := webrtc.SettingEngine{}
	se.DetachDataChannels()
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create PeerConnection: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	connected := make(chan struct{}, 1)
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[Tunnel] PeerConnection state: %s", state.String())
		if state == webrtc.PeerConnectionStateConnected {
			// ICE candidate pair 정보 로깅
			stats := pc.GetStats()
			var nominatedPair *webrtc.ICECandidatePairStats
			candidates := make(map[string]webrtc.ICECandidateStats)

			for _, s := range stats {
				if pair, ok := s.(webrtc.ICECandidatePairStats); ok && pair.Nominated {
					nominatedPair = &pair
				}
				if cand, ok := s.(webrtc.ICECandidateStats); ok {
					candidates[cand.ID] = cand
				}
			}

			if nominatedPair != nil {
				local := candidates[nominatedPair.LocalCandidateID]
				remote := candidates[nominatedPair.RemoteCandidateID]
				log.Printf("[ICE] Selected Pair: %s <-> %s (RTT: %.1fms)",
					local.CandidateType, remote.CandidateType, nominatedPair.CurrentRoundTripTime*1000)
			}

			select {
			case connected <- struct{}{}:
			default:
			}
		}
	})

	// offer 생성 (DataChannel 하나 있어야 SDP에 data section 포함)
	initDC, err := pc.CreateDataChannel("init", nil)
	if err != nil {
		pc.Close()
		return nil, fmt.Errorf("failed to create init DataChannel: %w", err)
	}
	initDC.OnOpen(func() { initDC.Close() })

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		pc.Close()
		return nil, fmt.Errorf("failed to create offer: %w", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		pc.Close()
		return nil, fmt.Errorf("failed to set local description: %w", err)
	}

	// ICE gathering 완료 대기
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	select {
	case <-gatherComplete:
		log.Println("[Tunnel] ICE gathering complete")
	case <-connectCtx.Done():
		pc.Close()
		return nil, fmt.Errorf("ICE gathering timeout")
	}

	// Hub에 offer 전송, answer 수신
	log.Println("[Tunnel] Sending offer to Hub...")
	answer, err := t.HubClient.SignalConnect(ctx, t.Config.ContainerID, *pc.LocalDescription())
	if err != nil {
		pc.Close()
		return nil, fmt.Errorf("signaling failed: %w", err)
	}

	if err := pc.SetRemoteDescription(*answer); err != nil {
		pc.Close()
		return nil, fmt.Errorf("failed to set remote description: %w", err)
	}

	// Connected 대기
	select {
	case <-connected:
		log.Println("[Tunnel] PeerConnection connected")
		return pc, nil
	case <-connectCtx.Done():
		pc.Close()
		return nil, fmt.Errorf("PeerConnection connect timeout")
	}
}

func (t *Tunnel) acceptLoop(ctx context.Context, pc *webrtc.PeerConnection, ln net.Listener, portKey string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[Tunnel] Accept error on %s: %v", portKey, err)
			return
		}

		go t.handleConnection(ctx, pc, conn, portKey)
	}
}

func (t *Tunnel) handleConnection(ctx context.Context, pc *webrtc.PeerConnection, tcpConn net.Conn, portKey string) {
	dc, err := pc.CreateDataChannel(portKey, nil)
	if err != nil {
		log.Printf("[Tunnel] Failed to create DataChannel %s: %v", portKey, err)
		tcpConn.Close()
		return
	}

	dc.OnOpen(func() {
		rawDC, err := dc.Detach()
		if err != nil {
			log.Printf("[Tunnel] Detach failed for %s: %v", portKey, err)
			tcpConn.Close()
			return
		}

		log.Printf("[Tunnel] Relay started: %s <-> DataChannel(%s)", tcpConn.RemoteAddr(), portKey)

		go func() {
			io.Copy(rawDC, tcpConn) // TCP → DataChannel
			rawDC.Close()
			tcpConn.Close()
		}()
		go func() {
			io.Copy(tcpConn, rawDC) // DataChannel → TCP
			tcpConn.Close()
			rawDC.Close()
		}()
	})
}

// parseContainerPort extracts the port number from a key like "3306/tcp".
func parseContainerPort(key string) (int, error) {
	parts := strings.Split(key, "/")
	if len(parts) == 0 {
		return 0, fmt.Errorf("empty port key")
	}
	port, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid port number %q: %w", parts[0], err)
	}
	return port, nil
}
