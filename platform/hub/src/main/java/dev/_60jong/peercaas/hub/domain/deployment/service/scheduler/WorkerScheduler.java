package dev._60jong.peercaas.hub.domain.deployment.service.scheduler;

import dev._60jong.peercaas.hub.domain.agent.model.AgentStatus;
import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;
import dev._60jong.peercaas.hub.domain.agent.repository.WorkerAgentRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.time.LocalDateTime;
import java.util.Comparator;
import java.util.List;
import java.util.Optional;

@Slf4j
@RequiredArgsConstructor
@Service
public class WorkerScheduler {

    private final WorkerAgentRepository workerAgentRepository;

    /**
     * 적절한 워커를 선정합니다.
     * 1. 리소스 조건 및 하트비트 상태 필터링
     * 2. 지연시간(Latency) 및 리소스 여유율 기반 스코어링
     */
    public Optional<WorkerAgent> selectBestWorker(Double requiredCpu, Long requiredMemoryMb) {
        LocalDateTime threshold = LocalDateTime.now().minusSeconds(30); // 30초 이내 하트비트

        List<WorkerAgent> candidates = workerAgentRepository.findAvailableWorkers(
                AgentStatus.ACTIVE,
                threshold,
                requiredCpu,
                requiredMemoryMb
        );

        if (candidates.isEmpty()) {
            log.warn("No available workers found for CPU: {}, Memory: {}", requiredCpu, requiredMemoryMb);
            return Optional.empty();
        }

        // 스코어링 로직: 지연시간이 낮고 리소스 여유가 많은 순
        return candidates.stream()
                .max(Comparator.comparingDouble(this::calculateScore));
    }

    /**
     * 워커 점수 계산 (높을수록 좋음)
     * Score = (1 / Latency) * W1 + (AvailableMemory / TotalMemory) * W2
     */
    private double calculateScore(WorkerAgent worker) {
        double latencyScore = (worker.getAverageLatencyMs() > 0) ? (1000.0 / worker.getAverageLatencyMs()) : 1000.0;
        double resourceScore = (double) worker.getAvailableMemoryMb() / worker.getTotalMemoryMb() * 100.0;

        // 가중치 적용 (네트워크 70%, 리소스 30%)
        return (latencyScore * 0.7) + (resourceScore * 0.3);
    }
}
