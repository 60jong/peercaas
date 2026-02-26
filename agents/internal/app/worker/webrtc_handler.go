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

// ConnectWebRTCPayload: Client-agent → Worker (요청)
type ConnectWebRTCPayload struct {
	ContainerID string                    `json:"containerId"`
	Offer       webrtc.SessionDescription `json:"offer"`
	ReplyQueue  string                    `json:"replyQueue"`
}

// ConnectWebRTCAnswerPayload: Worker → Client-agent (응답)
type ConnectWebRTCAnswerPayload struct {
	ContainerID string                    `json:"containerId"`
	Answer      webrtc.SessionDescription `json:"answer"`
}

type ConnectWebRTCHandler struct {
	Store        *ContainerStore
	Broker       core.Broker
	WorkerID     string
	DockerCli    *client.Client
	Traffic      *metrics.TrafficStore
	WebRTCConfig *webrtc.Configuration // nil이면 기본값(STUN) 사용
}

func (h *ConnectWebRTCHandler) Handle(ctx context.Context, msg core.CommandMessage) error {
	traceId := msg.TraceID

	var p ConnectWebRTCPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("invalid CONNECT_WEBRTC payload: %w", err)
	}

	log.Printf(">> [WEBRTC] TraceID: %s, ContainerID: %s", traceId, p.ContainerID)

	// 1. Store에서 컨테이너 정보 조회 (없으면 Docker inspect로 복구)
	info, ok := h.Store.Get(p.ContainerID)
	if !ok {
		log.Printf(">> [WEBRTC] Container not in store, falling back to Docker inspect: %s", p.ContainerID)
		recovered, err := h.recoverFromDocker(ctx, p.ContainerID)
		if err != nil {
			return fmt.Errorf("container not found in store and Docker inspect failed: %w", err)
		}
		h.Store.Put(recovered)
		info = recovered
		log.Printf(">> [WEBRTC] Container recovered from Docker: %s (ports: %v)", p.ContainerID, info.PortBindings)
	}

	// 2. PeerConnection 생성 (DetachDataChannels 활성화)
	se := webrtc.SettingEngine{}
	se.DetachDataChannels()

	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	iceConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	if h.WebRTCConfig != nil {
		iceConfig = *h.WebRTCConfig
	}

	pc, err := api.NewPeerConnection(iceConfig)
	if err != nil {
		return fmt.Errorf("failed to create PeerConnection: %w", err)
	}

	// 3. DataChannel 핸들러 등록 (TCP 릴레이)
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		label := dc.Label()
		log.Printf(">> [WEBRTC] DataChannel opened: %s (ContainerID: %s)", label, p.ContainerID)

		dc.OnOpen(func() {
			hostPort, exists := info.PortBindings[label]
			if !exists {
				log.Printf(">> [WEBRTC] Unknown port label: %s", label)
				dc.Close()
				return
			}

			dcSocket, err := dc.Detach()
			if err != nil {
				log.Printf(">> [WEBRTC] Detach failed: %v", err)
				return
			}

			// 컨테이너의 호스트 포트로 TCP 연결
			addr := fmt.Sprintf("127.0.0.1:%d", hostPort)
			tcpConn, err := net.Dial("tcp", addr)
			if err != nil {
				log.Printf(">> [WEBRTC] TCP connect failed (%s): %v", addr, err)
				dcSocket.Close()
				return
			}

			log.Printf(">> [WEBRTC] TCP relay started: %s <-> %s", label, addr)

			// 트래픽 집계 후 양방향 릴레이 (dcSocket=remote, tcpConn=local/container)
			stats := h.Traffic.GetOrCreate(p.ContainerID, "webrtc")
			stats.IncrConn()
			go metrics.BridgeWithTraffic(dcSocket, tcpConn, stats)
		})
	})

	// 4. 연결 상태 모니터링
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf(">> [WEBRTC] Connection state: %s (ContainerID: %s)", state.String(), p.ContainerID)
		if state == webrtc.PeerConnectionStateDisconnected || state == webrtc.PeerConnectionStateFailed {
			pc.Close()
		}
	})

	// 5. SDP 교환
	if err := pc.SetRemoteDescription(p.Offer); err != nil {
		pc.Close()
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		pc.Close()
		return fmt.Errorf("failed to create answer: %w", err)
	}

	if err := pc.SetLocalDescription(answer); err != nil {
		pc.Close()
		return fmt.Errorf("failed to set local description: %w", err)
	}

	// 6. ICE gathering 완료 대기 (30초 타임아웃)
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	select {
	case <-gatherComplete:
		// ICE gathering 완료
	case <-time.After(30 * time.Second):
		pc.Close()
		return fmt.Errorf("ICE gathering timeout")
	case <-ctx.Done():
		pc.Close()
		return fmt.Errorf("context cancelled during ICE gathering: %w", ctx.Err())
	}

	// 7. PeerConnection을 Store에 등록
	h.Store.AddPeerConnection(p.ContainerID, pc)

	// 8. Answer를 replyQueue에 발행
	answerPayload := ConnectWebRTCAnswerPayload{
		ContainerID: p.ContainerID,
		Answer:      *pc.LocalDescription(),
	}

	payloadBytes, err := json.Marshal(answerPayload)
	if err != nil {
		pc.Close()
		return fmt.Errorf("failed to marshal answer payload: %w", err)
	}

	replyMsg := core.CommandMessage{
		CmdType:   "CONNECT_WEBRTC_ANSWER",
		TraceID:   traceId,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	replyBytes, err := json.Marshal(replyMsg)
	if err != nil {
		pc.Close()
		return fmt.Errorf("failed to marshal reply message: %w", err)
	}

	publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.Broker.Publish(publishCtx, p.ReplyQueue, replyBytes); err != nil {
		pc.Close()
		return fmt.Errorf("failed to publish answer to %s: %w", p.ReplyQueue, err)
	}

	log.Printf(">> [WEBRTC] Answer published to %s (ContainerID: %s)", p.ReplyQueue, p.ContainerID)

	return nil
}

// recoverFromDocker Docker inspect로 ContainerInfo를 복구 (Worker 재시작 시 Store가 비었을 때)
func (h *ConnectWebRTCHandler) recoverFromDocker(ctx context.Context, containerID string) (*ContainerInfo, error) {
	if h.DockerCli == nil {
		return nil, fmt.Errorf("Docker client not available")
	}

	inspect, err := h.DockerCli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("Docker inspect failed: %w", err)
	}

	if !inspect.State.Running {
		return nil, fmt.Errorf("container %s is not running (state: %s)", containerID, inspect.State.Status)
	}

	portBindings := make(map[string]int)
	for containerPort, bindings := range inspect.NetworkSettings.Ports {
		if len(bindings) == 0 {
			continue
		}
		hostPort, err := strconv.Atoi(bindings[0].HostPort)
		if err != nil {
			log.Printf(">> [WEBRTC] Invalid host port for %s: %s", containerPort, bindings[0].HostPort)
			continue
		}
		portBindings[string(containerPort)] = hostPort
	}

	return &ContainerInfo{
		ContainerID:  containerID,
		Name:         inspect.Name,
		PortBindings: portBindings,
	}, nil
}
