package dev._60jong.peercaas.hub.domain.metrics.dto;

import java.util.List;

public record AgentSnapshot(
        String agentId,
        String agentType,        // "client" or "worker"
        String reportedAt,       // HH:mm:ss UTC
        boolean stale,           // true if last report > 15s ago
        List<ContainerTrafficInfo> containers
) {}
