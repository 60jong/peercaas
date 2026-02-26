package dev._60jong.peercaas.hub.domain.agent.consumer;

import dev._60jong.peercaas.hub.domain.agent.model.vo.WorkerHeartbeatPayload;
import dev._60jong.peercaas.hub.domain.agent.repository.WorkerAgentRepository;
import dev._60jong.peercaas.hub.domain.container.service.ContainerService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

@Slf4j
@Component
@RequiredArgsConstructor
public class WorkerHeartbeatConsumer {

    private final WorkerAgentRepository workerAgentRepository;
    private final ContainerService containerService;

    @Transactional
    @RabbitListener(queues = "peercaas.worker.heartbeat")
    public void handleHeartbeat(WorkerHeartbeatPayload payload) {
        log.debug("Received heartbeat from worker: {}", payload.getWorkerId());

        // 1. 워커 시스템 정보 업데이트
        workerAgentRepository.findByWorkerId(payload.getWorkerId())
                .ifPresentOrElse(
                        worker -> {
                            worker.updateHeartbeat(
                                    payload.getAvailableCpu(),
                                    payload.getAvailableMemoryMb(),
                                    payload.getAverageLatencyMs()
                            );
                        },
                        () -> log.warn("Heartbeat received from unregistered worker: {}", payload.getWorkerId())
                );

        // 2. 컨테이너 개별 메트릭 업데이트
        if (payload.getContainers() != null) {
            payload.getContainers().forEach(metric ->
                    containerService.updateMetrics(metric.getContainerId(), metric.getTxBytes(), metric.getRxBytes())
            );
        }
    }
}
