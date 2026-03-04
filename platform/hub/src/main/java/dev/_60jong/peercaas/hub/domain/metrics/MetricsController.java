package dev._60jong.peercaas.hub.domain.metrics;

import dev._60jong.peercaas.hub.domain.metrics.dto.AgentSnapshot;
import dev._60jong.peercaas.hub.domain.metrics.dto.ContainerMetricsResponse;
import dev._60jong.peercaas.hub.domain.metrics.dto.MetricsReport;
import dev._60jong.peercaas.hub.infra.victoriametrics.VictoriaMetricsClient;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.time.Instant;
import java.util.List;
import java.util.Map;

@RequiredArgsConstructor
@RestController
@RequestMapping("/api/v1/metrics")
public class MetricsController {

    private final MetricsStore metricsStore;
    private final VictoriaMetricsClient vmClient;

    /** Agents → Hub: 5초마다 트래픽 메트릭 보고 */
    @PostMapping
    public ResponseEntity<Void> receiveMetrics(@RequestBody MetricsReport report) {
        metricsStore.update(report);
        return ResponseEntity.ok().build();
    }

    /** Dashboard JS → Hub: key(agentId)에 해당하는 에이전트 메트릭 조회 */
    @GetMapping("/{key}")
    public ResponseEntity<AgentSnapshot> getByKey(@PathVariable String key) {
        return metricsStore.getByKey(key)
                .map(ResponseEntity::ok)
                .orElse(ResponseEntity.notFound().build());
    }

    /** 전체 메트릭 조회 (admin용) */
    @GetMapping
    public Map<String, List<AgentSnapshot>> getAll() {
        return metricsStore.getAll();
    }

    /**
     * 컨테이너별 시계열 메트릭 조회 (VictoriaMetrics 프록시)
     * range: 15m | 1h | 6h | 24h
     */
    @GetMapping("/container/{containerId}")
    public ResponseEntity<ContainerMetricsResponse> getContainerMetrics(
            @PathVariable String containerId,
            @RequestParam(defaultValue = "1h") String range
    ) {
        long end   = Instant.now().getEpochSecond();
        long start = end - rangeToSeconds(range);
        String step = rangeToStep(range);

        String label = "{container_id=\"" + containerId + "\"}";

        return ResponseEntity.ok(new ContainerMetricsResponse(
                vmClient.queryRange("container_usage_cpu_usage"    + label, start, end, step),
                vmClient.queryRange("container_usage_mem_usage_mb" + label, start, end, step),
                vmClient.queryRange("container_usage_net_tx_bytes" + label, start, end, step),
                vmClient.queryRange("container_usage_net_rx_bytes" + label, start, end, step)
        ));
    }

    private long rangeToSeconds(String range) {
        return switch (range) {
            case "15m" -> 15 * 60L;
            case "6h"  -> 6  * 3600L;
            case "24h" -> 24 * 3600L;
            default    -> 3600L;
        };
    }

    private String rangeToStep(String range) {
        return switch (range) {
            case "15m" -> "15s";
            case "6h"  -> "2m";
            case "24h" -> "10m";
            default    -> "30s";
        };
    }
}
