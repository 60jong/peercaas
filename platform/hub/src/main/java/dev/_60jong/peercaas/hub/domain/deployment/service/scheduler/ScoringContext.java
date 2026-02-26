package dev._60jong.peercaas.hub.domain.deployment.service.scheduler;

import lombok.Builder;
import lombok.Getter;

@Getter
@Builder
public class ScoringContext {

    private final String clientIpAddress;
    private final Double requiredCpu;
    private final Long requiredMemoryMb;
}
