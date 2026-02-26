package dev._60jong.peercaas.hub.domain.deployment.service.scheduler;

import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import java.util.List;

@Slf4j
@Component
@RequiredArgsConstructor
public class CompositeScoreCalculator {

    private final List<ScoringFactor> scoringFactors;
    private final ScoringWeightConfig weightConfig;

    /**
     * Composite Score 계산:
     * score = Σ(factor.score × weight) / Σ(weight) (enabled & applicable factors만)
     */
    public double calculate(WorkerAgent worker, ScoringContext context) {
        double weightedSum = 0.0;
        double totalWeight = 0.0;

        for (ScoringFactor factor : scoringFactors) {
            if (weightConfig.isDisabled(factor.key())) {
                continue;
            }
            if (!factor.isApplicable(context)) {
                continue;
            }

            double weight = weightConfig.getWeight(factor.key());
            if (weight <= 0.0) {
                continue;
            }

            double score = factor.score(worker, context);
            weightedSum += score * weight;
            totalWeight += weight;

            log.debug("[Scorer] Worker={}, Factor={}, Score={}, Weight={}",
                    worker.getWorkerId(), factor.key(), score, weight);
        }

        double compositeScore = totalWeight > 0 ? weightedSum / totalWeight : 0.0;
        log.debug("[Scorer] Worker={}, CompositeScore={}", worker.getWorkerId(), compositeScore);
        return compositeScore;
    }
}
