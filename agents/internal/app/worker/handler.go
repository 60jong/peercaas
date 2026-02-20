package worker

import "agents/internal/core"

// === [요청] 메인 페이로드 (Hub -> Worker) ===
// RabbitMQ payload 구조에 맞춤
type ContainerPayload struct {
	Registry      string            `json:"registry"`
	Image         string            `json:"image"`
	Tag           string            `json:"tag"`
	Name          string            `json:"name"` // 요청된 이름
	Command       []string          `json:"command,omitempty"`
	Ports         []PortMapping     `json:"ports"`
	Env           map[string]string `json:"env"`
	Volumes       []VolumeMapping   `json:"volumes"` // null 처리
	Resources     ResourceLimit     `json:"resources"`
	RestartPolicy string            `json:"restartPolicy"`
}

type PortMapping struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort"`
	Protocol      string `json:"protocol"`
}

type VolumeMapping struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly"`
}

type ResourceLimit struct {
	MemoryMB int64   `json:"memoryMb"`
	CPU      float64 `json:"cpu"`
}

// === 보상 트랜잭션 페이로드 ===
type DeleteContainerPayload struct {
	ContainerId string `json:"containerId"`
	Force       bool   `json:"force"`
}

// === 결과 페이로드 (Worker -> Hub) ===
type DeploymentResultPayload struct {
	WorkerId          string         `json:"workerId"`
	TraceId           string         `json:"traceId"`
	RequesterId       string         `json:"requesterId"`
	ResultStatus      string         `json:"resultStatus"`      // "SUCCESS" or "FAILED"
	ContainerId       string         `json:"containerId"`       // Docker Container ID
	HostContainerName string         `json:"hostContainerName"` // 실제 생성된 이름
	FailureReason     string         `json:"failureReason"`     // 실패 시 에러 메시지
	PortBindings      map[string]int `json:"portBindings"`      // ContainerPort -> HostPort
}

// ResultPublisher RabbitMQ Publisher Interface
type ResultPublisher interface {
	PublishResult(msg core.CommandMessage) error
}
