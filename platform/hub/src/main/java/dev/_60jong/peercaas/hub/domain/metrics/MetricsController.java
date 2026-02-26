package dev._60jong.peercaas.hub.domain.metrics;

import dev._60jong.peercaas.hub.domain.metrics.dto.AgentSnapshot;
import dev._60jong.peercaas.hub.domain.metrics.dto.MetricsReport;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

@RequiredArgsConstructor
@RestController
@RequestMapping("/api/v1/metrics")
public class MetricsController {

    private final MetricsStore metricsStore;

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
}
