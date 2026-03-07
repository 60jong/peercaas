package dev._60jong.peercaas.hub.domain.agent.consumer;

import dev._60jong.peercaas.hub.domain.agent.model.vo.WorkerHeartbeatPayload;
import dev._60jong.peercaas.hub.domain.agent.repository.WorkerAgentRepository;
import dev._60jong.peercaas.hub.domain.container.service.ContainerService;
import dev._60jong.peercaas.hub.domain.metrics.MetricsStore;
import dev._60jong.peercaas.hub.domain.metrics.dto.ContainerTrafficInfo;
import dev._60jong.peercaas.hub.domain.metrics.dto.MetricsReport;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

import java.time.Clock;
import java.time.Instant;
import java.util.Collections;
import java.util.List;
import java.util.stream.Collectors;

@Slf4j
@Component
@RequiredArgsConstructor
public class WorkerHeartbeatConsumer {

    private final WorkerAgentRepository workerAgentRepository;
    private final ContainerService containerService;
    private final MetricsStore metricsStore;
    private final Clock clock;

    @Transactional
    @RabbitListener(queues = "peercaas.worker.heartbeat")
    public void handleHeartbeat(WorkerHeartbeatPayload payload) {
        log.debug("Received heartbeat from worker: {}", payload.getWorkerId());

        workerAgentRepository.findByWorkerId(payload.getWorkerId())
                .ifPresentOrElse(
                        worker -> {
                            int containerCount = payload.getContainers() != null ? payload.getContainers().size() : 0;
                            worker.updateHeartbeat(
                                    payload.getAvailableCpu(),
                                    payload.getAvailableMemoryMb(),
                                    payload.getAverageLatencyMs(),
                                    containerCount,
                                    clock
                            );
                        },
                        () -> log.warn("Heartbeat received from unregistered worker: {}", payload.getWorkerId())
                );

        if (payload.getContainers() != null) {
            payload.getContainers().forEach(metric ->
                    containerService.updateMetrics(metric.getContainerId(), metric.getTxBytes(), metric.getRxBytes())
            );
        }

        // Store in MetricsStore for real-time dashboard display
        metricsStore.update(new MetricsReport(
                "worker",
                payload.getWorkerId(),
                Instant.now().getEpochSecond(),
                payload.getContainers() == null ? Collections.emptyList() :
                        payload.getContainers().stream()
                                .map(c -> new ContainerTrafficInfo(
                                        c.getContainerId(),
                                        "unknown", // Transport not in heartbeat payload currently
                                        c.getTxBytes(),
                                        c.getRxBytes(),
                                        0, // connCount not in heartbeat payload currently
                                        "",
                                        ""
                                ))
                                .collect(Collectors.toList())
        ));
    }
}
