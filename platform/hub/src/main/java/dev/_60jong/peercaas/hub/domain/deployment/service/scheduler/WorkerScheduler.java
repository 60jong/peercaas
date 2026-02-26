package dev._60jong.peercaas.hub.domain.deployment.service.scheduler;

import dev._60jong.peercaas.hub.domain.agent.model.AgentStatus;
import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;
import dev._60jong.peercaas.hub.domain.agent.repository.WorkerAgentRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.time.Clock;
import java.time.LocalDateTime;
import java.util.Comparator;
import java.util.List;
import java.util.Optional;

@Slf4j
@RequiredArgsConstructor
@Service
public class WorkerScheduler {

    private final WorkerAgentRepository workerAgentRepository;
    private final CompositeScoreCalculator scoreCalculator;
    private final Clock clock;

    public Optional<WorkerAgent> selectBestWorker(ScoringContext context) {
        LocalDateTime threshold = LocalDateTime.now(clock).minusSeconds(30);

        Double requiredCpu = context.getRequiredCpu() != null ? context.getRequiredCpu() : 0.0;
        Long requiredMemoryMb = context.getRequiredMemoryMb() != null ? context.getRequiredMemoryMb() : 0L;

        List<WorkerAgent> candidates = workerAgentRepository.findAvailableWorkers(
                AgentStatus.ACTIVE,
                threshold,
                requiredCpu,
                requiredMemoryMb
        );

        if (candidates.isEmpty()) {
            log.warn("[Scheduler] No available workers found for CPU: {}, Memory: {}", requiredCpu, requiredMemoryMb);
            return Optional.empty();
        }

        return candidates.stream()
                .max(Comparator.comparingDouble(worker -> scoreCalculator.calculate(worker, context)));
    }
}
