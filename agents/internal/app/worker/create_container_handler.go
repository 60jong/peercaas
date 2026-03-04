package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strings"
	"time"

	"agents/internal/core"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// CreateContainerHandler CREATE_CONTAINER 명령 핸들러
type CreateContainerHandler struct {
	DockerCli   *client.Client
	Publisher   ResultPublisher  // 결과 전송 인터페이스
	WorkerId    string           // 이 워커의 ID
	RequesterId string           // 요청자 ID (선택적, 메시지에서 가져올 수도 있음)
	Store       *ContainerStore  // 컨테이너 정보 저장소
}

func (h *CreateContainerHandler) Handle(ctx context.Context, msg core.CommandMessage) error {
	traceId := msg.TraceID

	log.Printf(">> Processing %s (TraceID: %s, Timestamp: %d)", msg.CmdType, traceId, msg.Timestamp)

	var p ContainerPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("invalid container payload: %w", err)
	}

	log.Printf(">> [DEPLOY START] TraceID: %s, Worker: %s, Image: %s:%s",
		traceId, h.WorkerId, p.Image, p.Tag)

	// 5. 입력 검증
	if err := h.validatePayload(&p); err != nil {
		h.sendFailure(traceId, "Invalid Request: "+err.Error())
		return err
	}

	// Context 취소 체크
	if err := ctx.Err(); err != nil {
		h.sendFailure(traceId, "Context cancelled before start: "+err.Error())
		return err
	}

	// 6. 이미지 풀링
	fullImageName := h.buildFullImageName(p.Registry, p.Image, p.Tag)

	if err := h.pullImage(ctx, fullImageName); err != nil {
		h.sendFailure(traceId, "Image Pull Failed: "+err.Error())
		return err
	}

	// Context 취소 체크
	if err := ctx.Err(); err != nil {
		h.sendFailure(traceId, "Context cancelled after image pull: "+err.Error())
		return err
	}

	// 7. 포트 바인딩 설정
	exposedPorts, dockerPortBindings, err := h.configurePorts(p.Ports)
	if err != nil {
		h.sendFailure(traceId, "Port Config Failed: "+err.Error())
		return err
	}

	// 8. 리소스 설정
	hostConfig := &container.HostConfig{
		PortBindings: dockerPortBindings,
		Resources: container.Resources{
			Memory:   p.Resources.MemoryMB * 1024 * 1024, // MB -> Bytes
			NanoCPUs: int64(p.Resources.CPU * 1e9),       // Core -> NanoCPU
		},
		RestartPolicy: container.RestartPolicy{
			Name: h.normalizeRestartPolicy(p.RestartPolicy),
		},
	}

	// 볼륨 바인딩 추가 (있는 경우)
	if len(p.Volumes) > 0 {
		hostConfig.Binds = h.configureVolumes(p.Volumes)
	}

	// 9. 컨테이너 생성 (이름 충돌 회피 로직 포함)
	containerID, actualName, err := h.createContainerWithRetry(ctx, p, fullImageName, exposedPorts, hostConfig)
	if err != nil {
		h.sendFailure(traceId, "Container Create Failed: "+err.Error())
		return err
	}

	// Context 취소 체크
	if err := ctx.Err(); err != nil {
		h.sendFailure(traceId, "Context cancelled after container create: "+err.Error())
		// 생성된 컨테이너 정리
		_ = h.DockerCli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
		return err
	}

	// 10. 컨테이너 시작
	if err := h.DockerCli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		h.sendFailure(traceId, "Container Start Failed: "+err.Error())
		// 시작 실패 시 생성된 컨테이너 정리
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = h.DockerCli.ContainerRemove(cleanupCtx, containerID, container.RemoveOptions{Force: true})
		return err
	}

	// 11. 컨테이너 시작 확인 (헬스체크)
	if err := h.waitForContainerRunning(ctx, containerID); err != nil {
		h.sendFailure(traceId, "Container Health Check Failed: "+err.Error())
		// 헬스체크 실패 시 컨테이너 정리
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = h.DockerCli.ContainerRemove(cleanupCtx, containerID, container.RemoveOptions{Force: true})
		return err
	}

	// 12. 성공 응답 전송
	log.Printf(">> [DEPLOY SUCCESS] TraceID: %s, ContainerID: %s, Name: %s",
		traceId, containerID, actualName)

	portBindings := make(map[string]int, len(p.Ports))
	for _, m := range p.Ports {
		key := fmt.Sprintf("%d/%s", m.ContainerPort, m.Protocol)
		portBindings[key] = m.HostPort
	}

	// Extract ClientKey from Env for metrics aggregation
	clientKey := ""
	if v, ok := p.Env["PEERCAAS_CLIENT_KEY"]; ok {
		clientKey = v
	}

	// Store에 컨테이너 정보 등록
	if h.Store != nil {
		h.Store.Put(&ContainerInfo{
			ContainerID:  containerID,
			Name:         actualName,
			ClientKey:    clientKey,
			PortBindings: portBindings,
			TraceID:      traceId,
		})
		log.Printf("   - Container registered in store: %s (%s, clientKey: %s)", containerID, actualName, clientKey)
	}

	response := DeploymentResultPayload{
		WorkerId:          h.WorkerId,
		TraceId:           traceId,
		RequesterId:       h.RequesterId,
		ResultStatus:      "SUCCESS",
		ContainerId:       containerID,
		HostContainerName: actualName,
		PortBindings:      portBindings,
	}

	return h.publishResult(traceId, "DEPLOYMENT_RESULT", response)
}

// --- Helper Functions for Create ---

// normalizeRestartPolicy RestartPolicy 정규화 (대소문자 처리)
func (h *CreateContainerHandler) normalizeRestartPolicy(policy string) container.RestartPolicyMode {
	// RabbitMQ에서 "Always"로 올 수 있으므로 lowercase로 변환
	normalized := strings.ToLower(policy)

	switch normalized {
	case "always":
		return container.RestartPolicyAlways
	case "on-failure":
		return container.RestartPolicyOnFailure
	case "unless-stopped":
		return container.RestartPolicyUnlessStopped
	default:
		return container.RestartPolicyDisabled // "no" or empty
	}
}

// configureVolumes 볼륨 바인딩 설정
func (h *CreateContainerHandler) configureVolumes(volumes []VolumeMapping) []string {
	binds := make([]string, 0, len(volumes))
	for _, v := range volumes {
		bind := fmt.Sprintf("%s:%s", v.HostPath, v.ContainerPath)
		if v.ReadOnly {
			bind += ":ro"
		}
		binds = append(binds, bind)
	}
	return binds
}

// validatePayload 페이로드 유효성 검증
func (h *CreateContainerHandler) validatePayload(p *ContainerPayload) error {
	if p.Image == "" {
		return fmt.Errorf("image is required")
	}
	if p.Tag == "" {
		return fmt.Errorf("tag is required")
	}
	if p.Name == "" {
		return fmt.Errorf("container name is required")
	}

	// 리소스 검증
	if p.Resources.MemoryMB <= 0 {
		return fmt.Errorf("memory must be positive, got: %d", p.Resources.MemoryMB)
	}
	if p.Resources.CPU <= 0 {
		return fmt.Errorf("cpu must be positive, got: %f", p.Resources.CPU)
	}

	// 최대값 체크 (필요시 조정)
	const maxMemoryMB = 32 * 1024 // 32GB
	const maxCPU = 16.0           // 16 cores
	if p.Resources.MemoryMB > maxMemoryMB {
		return fmt.Errorf("memory exceeds maximum allowed: %d > %d", p.Resources.MemoryMB, maxMemoryMB)
	}
	if p.Resources.CPU > maxCPU {
		return fmt.Errorf("cpu exceeds maximum allowed: %f > %f", p.Resources.CPU, maxCPU)
	}

	// RestartPolicy 검증 (대소문자 무시)
	if p.RestartPolicy != "" {
		normalized := strings.ToLower(p.RestartPolicy)
		validRestartPolicies := map[string]bool{
			"no":             true,
			"always":         true,
			"on-failure":     true,
			"unless-stopped": true,
		}
		if !validRestartPolicies[normalized] {
			return fmt.Errorf("invalid restart policy: %s (allowed: no, always, on-failure, unless-stopped)", p.RestartPolicy)
		}
	}

	// 포트 검증
	for i, port := range p.Ports {
		if port.ContainerPort <= 0 || port.ContainerPort > 65535 {
			return fmt.Errorf("invalid container port at index %d: %d", i, port.ContainerPort)
		}
		if port.HostPort <= 0 || port.HostPort > 65535 {
			return fmt.Errorf("invalid host port at index %d: %d", i, port.HostPort)
		}
		if port.Protocol != "" && port.Protocol != "tcp" && port.Protocol != "udp" {
			return fmt.Errorf("invalid protocol at index %d: %s (allowed: tcp, udp)", i, port.Protocol)
		}
	}

	return nil
}

// buildFullImageName 전체 이미지 이름 생성
func (h *CreateContainerHandler) buildFullImageName(registry, image, tag string) string {
	fullImageName := fmt.Sprintf("%s:%s", image, tag)
	if registry != "" && registry != "docker.io" {
		fullImageName = fmt.Sprintf("%s/%s:%s", registry, image, tag)
	}
	return fullImageName
}

// pullImage 이미지 Pull (에러 처리 개선)
func (h *CreateContainerHandler) pullImage(ctx context.Context, imageName string) error {
	log.Printf("   - Pulling image: %s", imageName)

	reader, err := h.DockerCli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("image pull request failed: %w", err)
	}
	defer reader.Close()

	// Pull 진행 상황을 읽으면서 에러 체크
	buf := make([]byte, 4096)
	for {
		// Context 취소 체크
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("image pull cancelled: %w", err)
		}

		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to download image: %w", err)
		}

		// 진행 상황 로깅 (선택적)
		// log.Printf("   - Downloaded %d bytes", n)
		_ = n
	}

	log.Printf("   - Image pulled successfully: %s", imageName)
	return nil
}

// configurePorts 포트 바인딩 설정
func (h *CreateContainerHandler) configurePorts(mappings []PortMapping) (nat.PortSet, nat.PortMap, error) {
	exposed := nat.PortSet{}
	bindings := nat.PortMap{}

	for _, m := range mappings {
		protocol := m.Protocol
		if protocol == "" {
			protocol = "tcp"
		}

		// Docker Port Key: "80/tcp"
		p, err := nat.NewPort(protocol, fmt.Sprintf("%d", m.ContainerPort))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid port %d/%s: %w", m.ContainerPort, protocol, err)
		}

		exposed[p] = struct{}{}
		bindings[p] = []nat.PortBinding{
			{HostPort: fmt.Sprintf("%d", m.HostPort)},
		}
	}

	return exposed, bindings, nil
}

// createContainerWithRetry 이름 충돌 시 재시도 로직 (개선)
func (h *CreateContainerHandler) createContainerWithRetry(
	ctx context.Context,
	p ContainerPayload,
	imageName string,
	exposed nat.PortSet,
	hostConfig *container.HostConfig,
) (string, string, error) {

	// 환경 변수 리스트 생성
	envList := make([]string, 0, len(p.Env))
	for k, v := range p.Env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	config := &container.Config{
		Image:        imageName,
		ExposedPorts: exposed,
		Env:          envList,
		Cmd:          p.Command,
	}

	// 동시성 문제 해결을 위해 랜덤 suffix 추가
	rand.Seed(time.Now().UnixNano())

	targetName := p.Name
	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		// Context 취소 체크
		if err := ctx.Err(); err != nil {
			return "", "", fmt.Errorf("container create cancelled: %w", err)
		}

		log.Printf("   - Attempting to create container with name: %s (attempt %d/%d)", targetName, i+1, maxRetries)

		resp, err := h.DockerCli.ContainerCreate(ctx, config, hostConfig, &network.NetworkingConfig{}, nil, targetName)
		if err == nil {
			log.Printf("   - Container created successfully: %s (ID: %s)", targetName, resp.ID)
			return resp.ID, targetName, nil
		}

		// 이름 충돌 에러가 아니면 즉시 리턴
		if !strings.Contains(err.Error(), "Conflict") && !strings.Contains(err.Error(), "is already in use") {
			return "", "", fmt.Errorf("container create failed: %w", err)
		}

		// 충돌 시 이름 변경 후 재시도
		if i < 5 {
			targetName = fmt.Sprintf("%s-%d", p.Name, i+1)
		} else {
			// 5번 이상 실패 시 랜덤 suffix 추가
			randomSuffix := rand.Intn(10000)
			targetName = fmt.Sprintf("%s-%d-%d", p.Name, i+1, randomSuffix)
		}

		log.Printf("   - Name conflict detected, retrying with new name: %s", targetName)
	}

	return "", "", fmt.Errorf("failed to generate unique container name after %d retries", maxRetries)
}

// waitForContainerRunning 컨테이너가 실제로 running 상태가 될 때까지 대기
func (h *CreateContainerHandler) waitForContainerRunning(ctx context.Context, containerID string) error {
	timeout := 30 * time.Second
	checkInterval := 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for container to start: %w", ctx.Err())
		case <-ticker.C:
			inspect, err := h.DockerCli.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			if inspect.State.Running {
				log.Printf("   - Container is running: %s", containerID)
				return nil
			}

			// 컨테이너가 에러 상태로 종료된 경우
			if inspect.State.Dead || inspect.State.OOMKilled {
				return fmt.Errorf("container died: Dead=%v, OOMKilled=%v, ExitCode=%d, Error=%s",
					inspect.State.Dead, inspect.State.OOMKilled, inspect.State.ExitCode, inspect.State.Error)
			}

			// 아직 시작 중이면 계속 대기
			log.Printf("   - Waiting for container to start... (Status: %s)", inspect.State.Status)
		}
	}
}

// sendFailure 실패 응답 전송
func (h *CreateContainerHandler) sendFailure(traceId string, reason string) {
	log.Printf(">> [DEPLOY FAILED] TraceID: %s, Worker: %s, Reason: %s",
		traceId, h.WorkerId, reason)

	response := DeploymentResultPayload{
		WorkerId:      h.WorkerId,
		TraceId:       traceId,
		RequesterId:   h.RequesterId,
		ResultStatus:  "FAILED",
		FailureReason: reason,
	}

	if err := h.publishResult(traceId, "DEPLOYMENT_RESULT", response); err != nil {
		log.Printf("   - Failed to publish failure result: %v", err)
	}
}

// publishResult payload를 CommandMessage로 감싸서 발행
func (h *CreateContainerHandler) publishResult(traceId string, cmdType string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := core.CommandMessage{
		CmdType:   cmdType,
		TraceID:   traceId,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	return h.Publisher.PublishResult(msg)
}
