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

// Tunnel manages a WebRTC PeerConnection and bridges local TCP traffic through it.
type Tunnel struct {
	Config    *Config
	HubClient *HubClient
}

// Start runs the tunnel in standalone mode (WebRTC only).
func (t *Tunnel) Start(ctx context.Context) error {
	info, err := t.HubClient.GetContainerInfo(ctx, t.Config.ContainerID)
	if err != nil {
		return fmt.Errorf("failed to get container info: %w", err)
	}

	log.Printf("[Tunnel] Container %s is RUNNING on worker %s", info.ContainerID[:12], info.WorkerID)

	pc, err := t.Connect(ctx)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer pc.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[Tunnel] WebRTC state: %s", state)
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateDisconnected || state == webrtc.PeerConnectionStateClosed {
			cancel()
		}
	})

	var wg sync.WaitGroup
	for key := range info.PortBindings {
		port, err := parseContainerPort(key)
		if err != nil {
			log.Printf("[Tunnel] Skipping invalid port %q: %v", key, err)
			continue
		}

		addr := fmt.Sprintf("0.0.0.0:%d", port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("[Tunnel] Failed to listen on %s: %v", addr, err)
			continue
		}
		defer ln.Close()

		log.Printf("[Tunnel] Listening on %s for port %s", addr, key)

		wg.Add(1)
		go func(l net.Listener, k string) {
			defer wg.Done()
			t.acceptLoop(ctx, pc, l, k)
		}(ln, key)
	}

	<-ctx.Done()
	log.Println("[Tunnel] Shutting down...")
	wg.Wait()
	return nil
}

// Connect establishes a WebRTC PeerConnection via the Hub's signaling channel.
func (t *Tunnel) Connect(ctx context.Context) (*webrtc.PeerConnection, error) {
	se := webrtc.SettingEngine{}
	se.DetachDataChannels()
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})
	if err != nil {
		return nil, err
	}

	connected := make(chan struct{}, 1)
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateConnected {
			select {
			case connected <- struct{}{}:
			default:
			}
		}
	})

	// Create an initial data channel to ensure SDP includes data sections
	if _, err := pc.CreateDataChannel("init", nil); err != nil {
		_ = pc.Close()
		return nil, err
	}

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		_ = pc.Close()
		return nil, err
	}

	gatherCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	select {
	case <-webrtc.GatheringCompletePromise(pc):
	case <-gatherCtx.Done():
		_ = pc.Close()
		return nil, fmt.Errorf("ICE gathering timed out")
	}

	answer, err := t.HubClient.SignalConnect(ctx, t.Config.ContainerID, *pc.LocalDescription())
	if err != nil {
		_ = pc.Close()
		return nil, err
	}

	if err := pc.SetRemoteDescription(*answer); err != nil {
		_ = pc.Close()
		return nil, err
	}

	select {
	case <-connected:
		return pc, nil
	case <-time.After(15 * time.Second):
		_ = pc.Close()
		return nil, fmt.Errorf("timed out waiting for PeerConnection to connect")
	case <-ctx.Done():
		_ = pc.Close()
		return nil, ctx.Err()
	}
}

func (t *Tunnel) acceptLoop(ctx context.Context, pc *webrtc.PeerConnection, ln net.Listener, portKey string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[Tunnel] Accept error: %v", err)
			return
		}
		go t.bridgeConnection(ctx, pc, conn, portKey)
	}
}

func (t *Tunnel) bridgeConnection(ctx context.Context, pc *webrtc.PeerConnection, tcpConn net.Conn, portKey string) {
	defer tcpConn.Close()

	dc, err := pc.CreateDataChannel(portKey, nil)
	if err != nil {
		return
	}

	opened := make(chan struct{})
	dc.OnOpen(func() { close(opened) })

	select {
	case <-opened:
		raw, err := dc.Detach()
		if err != nil {
			return
		}
		defer raw.Close()

		done := make(chan struct{}, 2)
		go func() { _, _ = io.Copy(raw, tcpConn); done <- struct{}{} }()
		go func() { _, _ = io.Copy(tcpConn, raw); done <- struct{}{} }()
		<-done
	case <-time.After(10 * time.Second):
		_ = dc.Close()
	case <-ctx.Done():
		_ = dc.Close()
	}
}

func parseContainerPort(key string) (int, error) {
	parts := strings.Split(key, "/")
	if len(parts) == 0 {
		return 0, fmt.Errorf("empty port key")
	}
	return strconv.Atoi(parts[0])
}
