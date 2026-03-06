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

// RelayConnectHandler manages relay-based connections to containers.
type RelayConnectHandler struct {
	Store     *ContainerStore
	DockerCli *client.Client
	Traffic   *metrics.TrafficStore
}

// Handle establishes one or more relay tunnels based on the provided sessions.
func (h *RelayConnectHandler) Handle(ctx context.Context, msg core.CommandMessage) error {
	var p RelayConnectPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal RELAY_CONNECT payload: %w", err)
	}

	log.Printf("[Relay] Establishing tunnels for container %s via %s:%d", p.ContainerID[:12], p.RelayHost, p.RelayPort)

	info, ok := h.Store.Get(p.ContainerID)
	if !ok {
		log.Printf("[Relay] Container %s missing from store, attempting recovery", p.ContainerID[:12])
		recovered, err := h.recoverFromDocker(ctx, p.ContainerID)
		if err != nil {
			return fmt.Errorf("container recovery failed: %w", err)
		}
		h.Store.Add(recovered)
		info = recovered
	}

	stats := h.Traffic.GetOrCreate(p.ContainerID, "RELAY")
	for _, session := range p.Sessions {
		hostPort, exists := info.PortBindings[session.PortKey]
		if !exists {
			log.Printf("[Relay] Port mapping not found for key: %s", session.PortKey)
			continue
		}

		go func(s RelaySessionEntry, port int) {
			if err := h.runRelayBridge(p.RelayHost, p.RelayPort, s.Token, port, stats); err != nil {
				log.Printf("[Relay] Session %s closed: %v", s.Token[:8], err)
			}
		}(session, hostPort)
	}

	return nil
}

func (h *RelayConnectHandler) runRelayBridge(host string, port int, token string, localPort int, stats *metrics.ContainerTraffic) error {
	relayAddr := fmt.Sprintf("%s:%d", host, port)
	relayConn, err := net.DialTimeout("tcp", relayAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("relay dial failed for host %s: %w", host, err)
	}

	// Perform handshake by sending the session token
	if _, err := fmt.Fprintf(relayConn, "%s\n", token); err != nil {
		_ = relayConn.Close()
		return fmt.Errorf("handshake failed: %w", err)
	}

	containerAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
	containerConn, err := net.DialTimeout("tcp", containerAddr, 5*time.Second)
	if err != nil {
		_ = relayConn.Close()
		return fmt.Errorf("container dial failed: %w", err)
	}

	log.Printf("[Relay] Bridge established: %s <-> :%d", token[:8], localPort)
	stats.IncrConn()

	metrics.BridgeWithTraffic(relayConn, containerConn, stats)
	return nil
}

func (h *RelayConnectHandler) recoverFromDocker(ctx context.Context, containerID string) (*ContainerInfo, error) {
	// Re-use logic from WebRTC handler
	helper := &ConnectWebRTCHandler{DockerCli: h.DockerCli}
	return helper.recoverFromDocker(ctx, containerID)
}
