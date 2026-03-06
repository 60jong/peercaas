package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"agents/internal/core"
	"agents/internal/metrics"

	"github.com/docker/docker/client"
	"github.com/pion/webrtc/v3"
)

// ConnectWebRTCPayload defines the request from a client agent to establish a WebRTC tunnel.
type ConnectWebRTCPayload struct {
	ContainerID string                    `json:"containerId"`
	Offer       webrtc.SessionDescription `json:"offer"`
	ReplyQueue  string                    `json:"replyQueue"`
}

// ConnectWebRTCHandler manages the establishment of WebRTC peer connections.
// It bridges WebRTC data channels to local container TCP ports.
type ConnectWebRTCHandler struct {
	Store     *ContainerStore
	Broker    core.Broker
	WorkerID  string
	DockerCli *client.Client
	Traffic   *metrics.TrafficStore
}

// Handle processes a WebRTC connection request.
func (h *ConnectWebRTCHandler) Handle(ctx context.Context, msg core.CommandMessage) error {
	correlationID := msg.CorrelationID

	var p ConnectWebRTCPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal CONNECT_WEBRTC payload: %w", err)
	}

	log.Printf("[WebRTC] Establishing connection for container %s (CorrelationID: %s)", p.ContainerID[:12], correlationID)

	info, ok := h.Store.Get(p.ContainerID)
	if !ok {
		log.Printf("[WebRTC] Container %s missing from store, attempting recovery", p.ContainerID[:12])
		recovered, err := h.recoverFromDocker(ctx, p.ContainerID)
		if err != nil {
			return fmt.Errorf("failed to recover container info: %w", err)
		}
		h.Store.Add(recovered)
		info = recovered
	}

	se := webrtc.SettingEngine{}
	se.DetachDataChannels()

	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	pc, err := api.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create PeerConnection: %w", err)
	}

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		h.handleDataChannel(p.ContainerID, info, dc)
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[WebRTC] Connection state: %s (Container: %s)", state, p.ContainerID[:12])
		if state == webrtc.PeerConnectionStateDisconnected || state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			_ = pc.Close()
		}
	})

	if err := pc.SetRemoteDescription(p.Offer); err != nil {
		_ = pc.Close()
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		_ = pc.Close()
		return fmt.Errorf("failed to create answer: %w", err)
	}

	if err := pc.SetLocalDescription(answer); err != nil {
		_ = pc.Close()
		return fmt.Errorf("failed to set local description: %w", err)
	}

	gatherComplete := webrtc.GatheringCompletePromise(pc)
	select {
	case <-gatherComplete:
	case <-time.After(30 * time.Second):
		_ = pc.Close()
		return fmt.Errorf("ICE gathering timed out")
	case <-ctx.Done():
		_ = pc.Close()
		return ctx.Err()
	}

	h.Store.AddPeerConnection(p.ContainerID, pc)

	return h.sendAnswer(correlationID, p.ContainerID, p.ReplyQueue, *pc.LocalDescription())
}

func (h *ConnectWebRTCHandler) handleDataChannel(containerID string, info *ContainerInfo, dc *webrtc.DataChannel) {
	label := dc.Label()
	dc.OnOpen(func() {
		hostPort, exists := info.PortBindings[label]
		if !exists {
			log.Printf("[WebRTC] Invalid data channel label: %s", label)
			_ = dc.Close()
			return
		}

		dcSocket, err := dc.Detach()
		if err != nil {
			log.Printf("[WebRTC] Failed to detach data channel: %v", err)
			return
		}

		addr := fmt.Sprintf("127.0.0.1:%d", hostPort)
		tcpConn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			log.Printf("[WebRTC] TCP connection failed to %s: %v", addr, err)
			_ = dcSocket.Close()
			return
		}

		stats := h.Traffic.GetOrCreate(containerID, "WEBRTC")
		stats.IncrConn()
		go metrics.BridgeWithTraffic(dcSocket, tcpConn, stats)
	})
}

func (h *ConnectWebRTCHandler) sendAnswer(correlationID, containerID, replyQueue string, answer webrtc.SessionDescription) error {
	payload := WebRTCAnswerPayload{
		ContainerID: containerID,
		Answer:      json.RawMessage(mustMarshal(answer)),
	}

	payloadBytes, _ := json.Marshal(payload)
	replyMsg := core.CommandMessage{
		CmdType:       "CONNECT_WEBRTC_ANSWER",
		CorrelationID: correlationID,
		Payload:       payloadBytes,
		Timestamp:     time.Now().Unix(),
	}

	data, _ := json.Marshal(replyMsg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return h.Broker.Publish(ctx, replyQueue, data)
}

func (h *ConnectWebRTCHandler) recoverFromDocker(ctx context.Context, containerID string) (*ContainerInfo, error) {
	inspect, err := h.DockerCli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, err
	}

	bindings := make(map[string]int)
	for p, b := range inspect.NetworkSettings.Ports {
		if len(b) > 0 {
			port, _ := strconv.Atoi(b[0].HostPort)
			bindings[string(p)] = port
		}
	}

	return &ContainerInfo{
		ContainerID:  containerID,
		Name:         inspect.Name,
		PortBindings: bindings,
	}, nil
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
