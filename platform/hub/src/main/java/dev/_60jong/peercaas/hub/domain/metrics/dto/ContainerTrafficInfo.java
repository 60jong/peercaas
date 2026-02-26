package dev._60jong.peercaas.hub.domain.metrics.dto;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
public record ContainerTrafficInfo(
        String containerId,
        String transport,
        long txBytes,
        long rxBytes,
        int connCount,
        String startTime,
        String lastActive
) {}
