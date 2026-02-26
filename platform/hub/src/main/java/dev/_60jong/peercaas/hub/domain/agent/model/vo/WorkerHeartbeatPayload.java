package dev._60jong.peercaas.hub.domain.agent.model.vo;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.List;

@Getter
@NoArgsConstructor
@AllArgsConstructor
public class WorkerHeartbeatPayload {
    private String workerId;
    private Double availableCpu;
    private Long availableMemoryMb;
    private Double averageLatencyMs;
    private List<ContainerMetric> containers;

    @Getter
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ContainerMetric {
        private String containerId;
        private Long txBytes;
        private Long rxBytes;
    }
}
