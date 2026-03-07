package dev._60jong.peercaas.hub.domain.metrics.dto;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.List;

@Getter
@NoArgsConstructor
@AllArgsConstructor
public class WorkerMetricsResponse {
    private List<List<Object>> cpuUsage;
    private List<List<Object>> memUsage;
    private List<List<Object>> netTx;
    private List<List<Object>> netRx;
}
