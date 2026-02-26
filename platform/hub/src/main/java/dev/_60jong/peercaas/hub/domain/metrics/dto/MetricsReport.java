package dev._60jong.peercaas.hub.domain.metrics.dto;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

import java.util.List;

@JsonIgnoreProperties(ignoreUnknown = true)
public record MetricsReport(
        String agentType,
        String agentId,
        long timestamp,
        List<ContainerTrafficInfo> containers
) {}
