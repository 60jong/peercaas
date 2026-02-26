package dev._60jong.peercaas.hub.domain.deployment.service.scheduler.factor;

import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.ScoringContext;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.ScoringFactor;
import org.springframework.stereotype.Component;

@Component
public class LatencyScoringFactor implements ScoringFactor {

    private static final double MAX_LATENCY_MS = 500.0;

    @Override
    public String key() {
        return "latency";
    }

    @Override
    public double score(WorkerAgent worker, ScoringContext context) {
        double latency = worker.getAverageLatencyMs() != null ? worker.getAverageLatencyMs() : 0.0;
        return Math.max(0.0, 1.0 - latency / MAX_LATENCY_MS);
    }
}
