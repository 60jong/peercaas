// Package metrics provides collection, storage, and reporting of agent and container telemetry.
package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// ResourceStatus represents current worker resources and usage.
type ResourceStatus struct {
	AvailableCPU      float64
	AvailableMemoryMb int64
}

// StatsSnapshot represents usage for a specific container.
type StatsSnapshot struct {
	CPUUsage   float64
	MemUsageMb int64
}

// Collector manages system and container resource snapshots using the Docker API and system stats.
type Collector struct {
	maxCPU      float64
	maxMemoryMb int64
	dockerCli   *client.Client
}

// NewCollector creates a new resource monitoring instance.
func NewCollector(maxCPU float64, maxMemory int64, dockerCli *client.Client) *Collector {
	return &Collector{
		maxCPU:      maxCPU,
		maxMemoryMb: maxMemory,
		dockerCli:   dockerCli,
	}
}

// GetAvailableResources calculates real-time resource availability for the whole worker.
// It aggregates usage from all running Docker containers and the worker process itself.
func (c *Collector) GetAvailableResources(ctx context.Context) (ResourceStatus, error) {
	var totalContainerMemMb int64
	var totalContainerCPU float64

	containers, err := c.dockerCli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return ResourceStatus{}, fmt.Errorf("failed to list containers: %w", err)
	}

	for _, cn := range containers {
		s, err := c.GetContainerStats(ctx, cn.ID)
		if err != nil {
			// Log and continue if a single container fails to report stats
			continue
		}
		totalContainerMemMb += s.MemUsageMb
		totalContainerCPU += s.CPUUsage
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	workerUsedMemMb := int64(m.Alloc) / 1024 / 1024

	status := ResourceStatus{
		AvailableCPU:      c.maxCPU - totalContainerCPU,
		AvailableMemoryMb: c.maxMemoryMb - (workerUsedMemMb + totalContainerMemMb),
	}

	// Ensure we don't report negative availability
	if status.AvailableCPU < 0 {
		status.AvailableCPU = 0
	}
	if status.AvailableMemoryMb < 0 {
		status.AvailableMemoryMb = 0
	}

	return status, nil
}

// GetContainerStats fetches granular usage for a single container.
func (c *Collector) GetContainerStats(ctx context.Context, containerID string) (StatsSnapshot, error) {
	stats, err := c.dockerCli.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		return StatsSnapshot{}, fmt.Errorf("failed to get stats for container %s: %w", containerID, err)
	}
	defer stats.Body.Close()

	var v container.StatsResponse
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		return StatsSnapshot{}, fmt.Errorf("failed to decode container stats: %w", err)
	}

	var s StatsSnapshot
	s.MemUsageMb = int64(v.MemoryStats.Usage) / 1024 / 1024

	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		s.CPUUsage = (cpuDelta / systemDelta) * float64(v.CPUStats.OnlineCPUs)
	}

	return s, nil
}
