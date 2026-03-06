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

// DeleteContainerHandler handles DELETE_CONTAINER commands, typically used for compensation transactions.
// It ensures containers are stopped and removed, and associated resources in the store are cleaned up.
type DeleteContainerHandler struct {
	DockerCli *client.Client
	Store     *ContainerStore
}

// Handle executes the container removal logic.
func (h *DeleteContainerHandler) Handle(ctx context.Context, msg core.CommandMessage) error {
	var p DeleteContainerPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal delete payload: %w", err)
	}

	if p.ContainerId == "" {
		return fmt.Errorf("containerId is required")
	}

	log.Printf("[Handler] Deleting container %s (Force: %v)", p.ContainerId[:12], p.Force)

	// Clean up related metadata in the store
	if h.Store != nil {
		h.Store.Delete(p.ContainerId)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Check if container exists before attempting removal
	_, err := h.DockerCli.ContainerInspect(timeoutCtx, p.ContainerId)
	if err != nil {
		if client.IsErrNotFound(err) {
			log.Printf("[Handler] Container %s already removed or not found", p.ContainerId[:12])
			return nil // Idempotent success
		}
		// Log but continue to attempt removal anyway
		log.Printf("[Handler] Failed to inspect container %s: %v", p.ContainerId[:12], err)
	}

	// Attempt graceful stop if not forced
	if !p.Force {
		stopTimeout := 10
		if err := h.DockerCli.ContainerStop(timeoutCtx, p.ContainerId, container.StopOptions{Timeout: &stopTimeout}); err != nil {
			if !client.IsErrNotFound(err) && !strings.Contains(err.Error(), "is not running") {
				log.Printf("[Handler] Failed to stop container %s gracefully: %v", p.ContainerId[:12], err)
			}
		} else {
			log.Printf("[Handler] Container %s stopped gracefully", p.ContainerId[:12])
		}
	}

	// Remove the container and its volumes
	removeOptions := container.RemoveOptions{
		Force:         p.Force,
		RemoveVolumes: true,
	}

	if err := h.DockerCli.ContainerRemove(timeoutCtx, p.ContainerId, removeOptions); err != nil {
		if client.IsErrNotFound(err) || strings.Contains(err.Error(), "No such container") {
			return nil // Idempotent success
		}
		return fmt.Errorf("failed to remove container %s: %w", p.ContainerId, err)
	}

	log.Printf("[Handler] Successfully removed container %s", p.ContainerId[:12])
	return nil
}
