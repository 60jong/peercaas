package worker

import (
	"encoding/json"
)

// ContainerPayload defines the request structure from the Hub to create a container.
type ContainerPayload struct {
	Registry      string            `json:"registry"`
	Image         string            `json:"image"`
	Tag           string            `json:"tag"`
	Name          string            `json:"name"`
	Command       []string          `json:"command,omitempty"`
	Ports         []PortMapping     `json:"ports"`
	Env           map[string]string `json:"env"`
	Volumes       []VolumeMapping   `json:"volumes"`
	Resources     ResourceLimit     `json:"resources"`
	RestartPolicy string            `json:"restartPolicy"`
}

// PortMapping defines a single container-to-host port binding.
type PortMapping struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort"`
	Protocol      string `json:"protocol"`
}

// VolumeMapping defines a single container-to-host volume mount.
type VolumeMapping struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly"`
}

// ResourceLimit defines the compute constraints for a container.
type ResourceLimit struct {
	MemoryMB int64   `json:"memoryMb"`
	CPU      float64 `json:"cpu"`
}

// DeleteContainerPayload defines the request structure to remove a container.
type DeleteContainerPayload struct {
	ContainerId string `json:"containerId"`
	Force       bool   `json:"force"`
}

// DeploymentResultPayload defines the response sent back to the Hub after a deployment command.
type DeploymentResultPayload struct {
	WorkerId          string         `json:"workerId"`
	CorrelationId     string         `json:"correlationId"`
	RequesterId       string         `json:"requesterId"`
	ResultStatus      string         `json:"resultStatus"`      // "SUCCESS" or "FAILED"
	ContainerId       string         `json:"containerId"`       // Docker Container ID
	HostContainerName string         `json:"hostContainerName"` // Actual name generated
	FailureReason     string         `json:"failureReason"`     // Error message on failure
	PortBindings      map[string]int `json:"portBindings"`      // Mapping of "port/proto" to HostPort
}

// WebRTCAnswerPayload defines the response sent back to the Hub for WebRTC connection requests.
type WebRTCAnswerPayload struct {
	ContainerID string          `json:"containerId"`
	Answer      json.RawMessage `json:"answer"`
}

// RelayConnectPayload defines the configuration for establishing relay connections when P2P fails.
type RelayConnectPayload struct {
	ContainerID string              `json:"containerId"`
	RelayHost   string              `json:"relayHost"`
	RelayPort   int                 `json:"relayPort"`
	Sessions    []RelaySessionEntry `json:"sessions"`
}

// RelaySessionEntry maps a container port to a specific relay session token.
type RelaySessionEntry struct {
	PortKey string `json:"portKey"`
	Token   string `json:"token"`
}
