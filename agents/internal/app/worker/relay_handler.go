package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"agents/internal/core"
	"agents/internal/metrics"

	"github.com/docker/docker/client"
)

type RelayConnectPayload struct {
	ContainerID string              `json:"containerId"`
	RelayHost   string              `json:"relayHost"`
	RelayPort   int                 `json:"relayPort"`
	Sessions    []RelaySessionEntry `json:"sessions"`
}

type RelaySessionEntry struct {
	PortKey string `json:"portKey"`
	Token   string `json:"token"`
}

type RelayConnectHandler struct {
	Store     *ContainerStore
	DockerCli *client.Client
	Traffic   *metrics.TrafficStore
}

func (h *RelayConnectHandler) Handle(ctx context.Context, msg core.CommandMessage) error {
	var p RelayConnectPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("invalid RELAY_CONNECT payload: %w", err)
	}

	log.Printf(">> [RELAY] ContainerID: %s, relay: %s:%d, sessions: %d",
		p.ContainerID, p.RelayHost, p.RelayPort, len(p.Sessions))

	// Store 조회, 없으면 Docker inspect로 복구
	info, ok := h.Store.Get(p.ContainerID)
	if !ok {
		log.Printf(">> [RELAY] Container not in store, falling back to Docker inspect")
		recovered, err := h.recoverFromDocker(ctx, p.ContainerID)
		if err != nil {
			return fmt.Errorf("container not found: %w", err)
		}
		h.Store.Put(recovered)
		info = recovered
	}

	// 각 세션마다 고루틴으로 relay 연결 수립
	stats := h.Traffic.GetOrCreate(p.ContainerID, "relay")
	for _, sess := range p.Sessions {
		hostPort, exists := info.PortBindings[sess.PortKey]
		if !exists {
			log.Printf(">> [RELAY] Unknown portKey: %s (portBindings: %v)", sess.PortKey, info.PortBindings)
			continue
		}

		go func(s RelaySessionEntry, hp int) {
			// ctx 대신 Background 사용 — Worker가 재시작해도 relay 연결은 유지
			if err := runRelayBridge(p.RelayHost, p.RelayPort, s.Token, hp, stats); err != nil {
				log.Printf(">> [RELAY] Session %s ended: %v", s.Token, err)
			}
		}(sess, hostPort)
	}

	return nil
}

// runRelayBridge: Engine relay 서버에 연결 → 컨테이너 포트와 브릿지
func runRelayBridge(relayHost string, relayPort int, token string, containerPort int, stats *metrics.ContainerTraffic) error {
	relayAddr := fmt.Sprintf("%s:%d", relayHost, relayPort)

	relayConn, err := net.DialTimeout("tcp", relayAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to relay %s: %w", relayAddr, err)
	}

	// 핸드셰이크: 세션 토큰 전송
	if _, err := fmt.Fprintf(relayConn, "%s\n", token); err != nil {
		relayConn.Close()
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	// 컨테이너 포트로 TCP 연결
	containerAddr := fmt.Sprintf("127.0.0.1:%d", containerPort)
	containerConn, err := net.DialTimeout("tcp", containerAddr, 5*time.Second)
	if err != nil {
		relayConn.Close()
		return fmt.Errorf("failed to connect to container %s: %w", containerAddr, err)
	}

	log.Printf(">> [RELAY] Bridge active: token=%s ↔ container:%d", token, containerPort)
	stats.IncrConn()

	// 양방향 브릿지 (relayConn=remote/client측, containerConn=local/컨테이너측)
	metrics.BridgeWithTraffic(relayConn, containerConn, stats)
	return nil
}

func (h *RelayConnectHandler) recoverFromDocker(ctx context.Context, containerID string) (*ContainerInfo, error) {
	return (&ConnectWebRTCHandler{DockerCli: h.DockerCli}).recoverFromDocker(ctx, containerID)
}
