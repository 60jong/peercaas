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

// CreateContainerHandler processes CREATE_CONTAINER commands from the Hub.
// It pulls images, configures networking and resources, and manages the Docker container lifecycle.
type CreateContainerHandler struct {
	DockerCli   *client.Client
	Publisher   ResultPublisher
	WorkerID    string
	RequesterID string
	Store       *ContainerStore
}

// Handle executes the container creation logic.
func (h *CreateContainerHandler) Handle(ctx context.Context, msg core.CommandMessage) error {
	correlationID := msg.CorrelationID

	log.Printf("[Handler] Processing CREATE_CONTAINER (CorrelationID: %s)", correlationID)

	var p ContainerPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal container payload: %w", err)
	}

	log.Printf("[Handler] Deploying image %s:%s for worker %s", p.Image, p.Tag, h.WorkerID)

	if err := h.validatePayload(&p); err != nil {
		h.sendFailure(correlationID, "Validation failed: "+err.Error())
		return err
	}

	if err := ctx.Err(); err != nil {
		h.sendFailure(correlationID, "Context cancelled: "+err.Error())
		return err
	}

	fullImageName := h.buildFullImageName(p.Registry, p.Image, p.Tag)
	if err := h.pullImage(ctx, fullImageName); err != nil {
		h.sendFailure(correlationID, "Image pull failed: "+err.Error())
		return err
	}

	exposedPorts, dockerPortBindings, err := h.configurePorts(p.Ports)
	if err != nil {
		h.sendFailure(correlationID, "Port configuration failed: "+err.Error())
		return err
	}

	hostConfig := &container.HostConfig{
		PortBindings: dockerPortBindings,
		Resources: container.Resources{
			Memory:   p.Resources.MemoryMB * 1024 * 1024,
			NanoCPUs: int64(p.Resources.CPU * 1e9),
		},
		RestartPolicy: container.RestartPolicy{
			Name: h.normalizeRestartPolicy(p.RestartPolicy),
		},
	}

	if len(p.Volumes) > 0 {
		hostConfig.Binds = h.configureVolumes(p.Volumes)
	}

	containerID, actualName, err := h.createContainerWithRetry(ctx, p, fullImageName, exposedPorts, hostConfig)
	if err != nil {
		h.sendFailure(correlationID, "Container creation failed: "+err.Error())
		return err
	}

	// Ensure cleanup if startup fails or context is cancelled
	cleanupPerformed := false
	defer func() {
		if !cleanupPerformed && err != nil {
			log.Printf("[Handler] Cleaning up container %s due to error: %v", containerID, err)
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = h.DockerCli.ContainerRemove(cleanupCtx, containerID, container.RemoveOptions{Force: true})
		}
	}()

	if err := h.DockerCli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		h.sendFailure(correlationID, "Container start failed: "+err.Error())
		return err
	}

	if err := h.waitForContainerRunning(ctx, containerID); err != nil {
		h.sendFailure(correlationID, "Container health check failed: "+err.Error())
		return err
	}

	log.Printf("[Handler] Successfully deployed container %s (Name: %s)", containerID[:12], actualName)

	portBindings := make(map[string]int, len(p.Ports))
	for _, m := range p.Ports {
		key := fmt.Sprintf("%d/%s", m.ContainerPort, m.Protocol)
		portBindings[key] = m.HostPort
	}

	clientKey := p.Env["PEERCAAS_CLIENT_KEY"]

	if h.Store != nil {
		h.Store.Add(&ContainerInfo{
			ContainerID:   containerID,
			CorrelationID: correlationID,
			ClientKey:     clientKey,
			ContainerPort: p.Ports[0].ContainerPort, // Assuming primary port
			PublicPort:    p.Ports[0].HostPort,
		})
	}

	response := DeploymentResultPayload{
		WorkerId:          h.WorkerID,
		CorrelationId:     correlationID,
		RequesterId:       h.RequesterID,
		ResultStatus:      "SUCCESS",
		ContainerId:       containerID,
		HostContainerName: actualName,
		PortBindings:      portBindings,
	}

	cleanupPerformed = true // Mark as successful to skip defer cleanup
	return h.publishResult(correlationID, "DEPLOYMENT_RESULT", response)
}

func (h *CreateContainerHandler) normalizeRestartPolicy(policy string) container.RestartPolicyMode {
	switch strings.ToLower(policy) {
	case "always":
		return container.RestartPolicyAlways
	case "on-failure":
		return container.RestartPolicyOnFailure
	case "unless-stopped":
		return container.RestartPolicyUnlessStopped
	default:
		return container.RestartPolicyDisabled
	}
}

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

func (h *CreateContainerHandler) validatePayload(p *ContainerPayload) error {
	if p.Image == "" || p.Tag == "" || p.Name == "" {
		return fmt.Errorf("image, tag, and name are required fields")
	}
	if p.Resources.MemoryMB <= 0 || p.Resources.CPU <= 0 {
		return fmt.Errorf("resources (CPU and Memory) must be positive values")
	}
	return nil
}

func (h *CreateContainerHandler) buildFullImageName(registry, image, tag string) string {
	if registry == "" || registry == "docker.io" {
		return fmt.Sprintf("%s:%s", image, tag)
	}
	return fmt.Sprintf("%s/%s:%s", registry, image, tag)
}

func (h *CreateContainerHandler) pullImage(ctx context.Context, imageName string) error {
	log.Printf("[Handler] Pulling image: %s", imageName)
	reader, err := h.DockerCli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull request failed: %w", err)
	}
	defer reader.Close()

	// Consume the pull stream to ensure completion and check for errors
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull stream: %w", err)
	}

	return nil
}

func (h *CreateContainerHandler) configurePorts(mappings []PortMapping) (nat.PortSet, nat.PortMap, error) {
	exposed := nat.PortSet{}
	bindings := nat.PortMap{}

	for _, m := range mappings {
		proto := m.Protocol
		if proto == "" {
			proto = "tcp"
		}

		p, err := nat.NewPort(proto, fmt.Sprintf("%d", m.ContainerPort))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid port definition %d/%s: %w", m.ContainerPort, proto, err)
		}

		exposed[p] = struct{}{}
		bindings[p] = []nat.PortBinding{{HostPort: fmt.Sprintf("%d", m.HostPort)}}
	}

	return exposed, bindings, nil
}

func (h *CreateContainerHandler) createContainerWithRetry(ctx context.Context, p ContainerPayload, imageName string, exposed nat.PortSet, hostConfig *container.HostConfig) (string, string, error) {
	env := make([]string, 0, len(p.Env))
	for k, v := range p.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	config := &container.Config{
		Image:        imageName,
		ExposedPorts: exposed,
		Env:          env,
		Cmd:          p.Command,
	}

	targetName := p.Name
	const maxRetries = 5

	for i := 0; i < maxRetries; i++ {
		if err := ctx.Err(); err != nil {
			return "", "", err
		}

		resp, err := h.DockerCli.ContainerCreate(ctx, config, hostConfig, &network.NetworkingConfig{}, nil, targetName)
		if err == nil {
			return resp.ID, targetName, nil
		}

		if !strings.Contains(err.Error(), "Conflict") && !strings.Contains(err.Error(), "is already in use") {
			return "", "", fmt.Errorf("failed to create container: %w", err)
		}

		// Handle name conflict by appending a random suffix
		targetName = fmt.Sprintf("%s-%x", p.Name, rand.Int31())
		log.Printf("[Handler] Name conflict, retrying as: %s", targetName)
	}

	return "", "", fmt.Errorf("exhausted retries for unique container name")
}

func (h *CreateContainerHandler) waitForContainerRunning(ctx context.Context, containerID string) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			inspect, err := h.DockerCli.ContainerInspect(ctx, containerID)
			if err != nil {
				return err
			}

			if inspect.State.Running {
				return nil
			}

			if inspect.State.Dead || inspect.State.OOMKilled {
				return fmt.Errorf("container exited prematurely (Status: %s)", inspect.State.Status)
			}
		}
	}
}

func (h *CreateContainerHandler) sendFailure(correlationID, reason string) {
	log.Printf("[Handler] Deployment failed (CorrelationID: %s): %s", correlationID, reason)

	_ = h.Publisher.Publish(context.Background(), correlationID, "DEPLOYMENT_RESULT", DeploymentResultPayload{
		WorkerId:      h.WorkerID,
		CorrelationId: correlationID,
		RequesterId:   h.RequesterID,
		ResultStatus:  "FAILED",
		FailureReason: reason,
	})
}

func (h *CreateContainerHandler) publishResult(correlationID, cmdType string, payload any) error {
	return h.Publisher.Publish(context.Background(), correlationID, cmdType, payload)
}
