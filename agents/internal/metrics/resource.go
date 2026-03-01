package metrics

import (
	"context"
	"encoding/json"
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

// Collector manages system and container resource snapshots.
type Collector struct {
	maxCPU      float64
	maxMemoryMb int64
	dockerCli   *client.Client
}

// NewCollector creates a new resource monitoring instance.
func NewCollector(maxCPU float64, maxMemory int64, cli *client.Client) *Collector {
	return &Collector{
		maxCPU:      maxCPU,
		maxMemoryMb: maxMemory,
		dockerCli:   cli,
	}
}

// GetAvailableResources calculates real-time resource availability for the whole worker.
func (c *Collector) GetAvailableResources(ctx context.Context) (ResourceStatus, error) {
	var totalContainerMemMb int64
	var totalContainerCPU float64

	containers, err := c.dockerCli.ContainerList(ctx, container.ListOptions{})
	if err == nil {
		for _, cn := range containers {
			s, err := c.GetContainerStats(ctx, cn.ID)
			if err == nil {
				totalContainerMemMb += s.MemUsageMb
				totalContainerCPU += s.CPUUsage
			}
		}
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	workerUsedMemMb := int64(m.Alloc) / 1024 / 1024

	status := ResourceStatus{
		AvailableCPU:      c.maxCPU - totalContainerCPU,
		AvailableMemoryMb: c.maxMemoryMb - (workerUsedMemMb + totalContainerMemMb),
	}

	if status.AvailableCPU < 0 { status.AvailableCPU = 0 }
	if status.AvailableMemoryMb < 0 { status.AvailableMemoryMb = 0 }

	return status, nil
}

// GetContainerStats fetches granular usage for a single container.
func (c *Collector) GetContainerStats(ctx context.Context, containerID string) (StatsSnapshot, error) {
	stats, err := c.dockerCli.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		return StatsSnapshot{}, err
	}
	defer stats.Body.Close()

	var v container.StatsResponse
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		return StatsSnapshot{}, err
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
