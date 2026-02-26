package dev._60jong.peercaas.hub.domain.deployment.service.scheduler;

import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;

public interface ScoringFactor {

    /**
     * 이 factor의 고유 키 (yaml weights 키와 매칭)
     */
    String key();

    /**
     * 워커의 점수를 계산합니다. [0.0, 1.0] 범위로 정규화 (1.0이 최상)
     */
    double score(WorkerAgent worker, ScoringContext context);

    /**
     * 현재 컨텍스트에서 이 factor가 적용 가능한지 판단
     */
    default boolean isApplicable(ScoringContext context) {
        return true;
    }
}
