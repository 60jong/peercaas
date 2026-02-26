package dev._60jong.peercaas.hub.domain.deployment.service.scheduler.factor;

import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.ScoringContext;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.ScoringFactor;
import org.springframework.stereotype.Component;

@Component
public class ResourceScoringFactor implements ScoringFactor {

    @Override
    public String key() {
        return "resource";
    }

    @Override
    public double score(WorkerAgent worker, ScoringContext context) {
        double cpuRatio = 0.0;
        if (worker.getTotalCpu() != null && worker.getTotalCpu() > 0) {
            double available = worker.getAvailableCpu() != null ? worker.getAvailableCpu() : 0.0;
            cpuRatio = available / worker.getTotalCpu();
        }

        double memRatio = 0.0;
        if (worker.getTotalMemoryMb() != null && worker.getTotalMemoryMb() > 0) {
            long available = worker.getAvailableMemoryMb() != null ? worker.getAvailableMemoryMb() : 0L;
            memRatio = (double) available / worker.getTotalMemoryMb();
        }

        return Math.min(1.0, (cpuRatio + memRatio) / 2.0);
    }
}
