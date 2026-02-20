package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"agents/internal/core"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// DeleteContainerHandler DELETE_CONTAINER 명령 핸들러 (보상 트랜잭션)
type DeleteContainerHandler struct {
	DockerCli *client.Client
	Store     *ContainerStore
}

func (h *DeleteContainerHandler) Handle(ctx context.Context, msg core.CommandMessage) error {
	var p DeleteContainerPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("invalid delete payload: %w", err)
	}

	log.Printf(">> [COMPENSATION] Deleting Container: %s (Force: %v)", p.ContainerId, p.Force)

	// PeerConnection 정리
	if h.Store != nil {
		h.Store.ClosePeerConnections(p.ContainerId)
	}

	// 입력 검증
	if p.ContainerId == "" {
		return fmt.Errorf("containerId is required")
	}

	// Timeout 설정
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 1. 컨테이너 상태 확인 (존재 여부)
	_, err := h.DockerCli.ContainerInspect(ctx, p.ContainerId)
	if err != nil {
		if client.IsErrNotFound(err) {
			log.Printf("   - Container already removed or not found: %s", p.ContainerId)
			return nil // 멱등성 보장
		}
		log.Printf("   - Failed to inspect container: %v", err)
		// Inspect 실패는 무시하고 삭제 시도
	}

	// 2. 컨테이너 중지 시도 (Force가 false인 경우 먼저 중지)
	if !p.Force {
		stopTimeout := 10 // 10초 대기
		if err := h.DockerCli.ContainerStop(ctx, p.ContainerId, container.StopOptions{
			Timeout: &stopTimeout,
		}); err != nil {
			// 이미 중지된 경우 무시
			if !strings.Contains(err.Error(), "is not running") {
				log.Printf("   - Failed to stop container (will try force remove): %v", err)
			}
		} else {
			log.Printf("   - Container stopped gracefully")
		}
	}

	// 3. 컨테이너 삭제
	options := container.RemoveOptions{
		Force:         p.Force,
		RemoveVolumes: true,
		RemoveLinks:   false,
	}

	err = h.DockerCli.ContainerRemove(ctx, p.ContainerId, options)
	if err != nil {
		// 이미 없는 컨테이너면 성공으로 간주 (멱등성)
		if client.IsErrNotFound(err) || strings.Contains(err.Error(), "No such container") {
			log.Printf("   - Container already removed or not found.")
			return nil
		}

		log.Printf("   - Failed to remove container: %v", err)
		return fmt.Errorf("failed to remove container %s: %w", p.ContainerId, err)
	}

	log.Printf("   - Container removed successfully: %s", p.ContainerId)

	// Store에서 컨테이너 정보 제거
	if h.Store != nil {
		h.Store.Delete(p.ContainerId)
		log.Printf("   - Container removed from store: %s", p.ContainerId)
	}

	return nil
}
