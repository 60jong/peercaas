package dev._60jong.peercaas.hub.domain.metrics.dto;

import lombok.AllArgsConstructor;
import lombok.Getter;

import java.util.List;

@Getter
@AllArgsConstructor
public class ContainerMetricsResponse {
    private final List<List<Object>> cpu;
    private final List<List<Object>> memory;
    private final List<List<Object>> txBytes;
    private final List<List<Object>> rxBytes;
}
