package dev._60jong.peercaas.hub.domain.deployment.controller.api.response;

import dev._60jong.peercaas.hub.domain.deployment.model.DeploymentStatus;
import lombok.AllArgsConstructor;
import lombok.Getter;

import java.time.LocalDateTime;

@Getter
@AllArgsConstructor
public class CreateDeploymentResponse {
    private Long deploymentId;      // DB PK (상태 조회용)
    private String traceId;         // 분산 추적 ID (로그 확인용)
    private DeploymentStatus status;// 현재 상태 (REQUESTED)
    private String workerId;        // 할당된 워커 ID
}
