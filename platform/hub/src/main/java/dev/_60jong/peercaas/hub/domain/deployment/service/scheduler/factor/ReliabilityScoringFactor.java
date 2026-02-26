package dev._60jong.peercaas.hub.domain.deployment.service.scheduler.factor;

import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;
import dev._60jong.peercaas.hub.domain.deployment.model.DeploymentStatus;
import dev._60jong.peercaas.hub.domain.deployment.repository.DeploymentRepository;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.ScoringContext;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.ScoringFactor;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Component;

import java.time.Clock;
import java.time.LocalDateTime;

@Component
@RequiredArgsConstructor
public class ReliabilityScoringFactor implements ScoringFactor {

    private final DeploymentRepository deploymentRepository;
    private final Clock clock;

    @Override
    public String key() {
        return "reliability";
    }

    @Override
    public double score(WorkerAgent worker, ScoringContext context) {
        LocalDateTime since = LocalDateTime.now(clock).minusHours(24);

        long totalCount = deploymentRepository.countByWorkerIdAndCreatedAtAfter(worker.getWorkerId(), since);
        if (totalCount == 0) {
            return 1.0;
        }

        long successCount = deploymentRepository.countByWorkerIdAndStatusAndCreatedAtAfter(
                worker.getWorkerId(), DeploymentStatus.RUNNING, since
        );

        return (double) successCount / totalCount;
    }
}
