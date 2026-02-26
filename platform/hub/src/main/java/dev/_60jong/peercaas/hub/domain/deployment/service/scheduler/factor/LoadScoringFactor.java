package dev._60jong.peercaas.hub.domain.deployment.service.scheduler.factor;

import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.ScoringContext;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.ScoringFactor;
import org.springframework.stereotype.Component;

@Component
public class LoadScoringFactor implements ScoringFactor {

    @Override
    public String key() {
        return "load";
    }

    @Override
    public double score(WorkerAgent worker, ScoringContext context) {
        int running = worker.getRunningContainerCount() != null ? worker.getRunningContainerCount() : 0;
        int max = worker.getMaxContainerCapacity() != null ? worker.getMaxContainerCapacity() : 20;

        if (max <= 0) {
            return 0.0;
        }

        return Math.max(0.0, 1.0 - (double) running / max);
    }
}
