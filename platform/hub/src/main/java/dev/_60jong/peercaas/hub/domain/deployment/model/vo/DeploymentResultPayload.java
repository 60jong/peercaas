package dev._60jong.peercaas.hub.domain.deployment.model.vo;

import lombok.AccessLevel;
import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.Map;

@Getter
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor
public class DeploymentResultPayload {

    /**
     * Worker Agent ID
     * ResultPayload 출처를 확인하기 위해 사용됨
     */
    private String workerId;

    /**
     * 트랜잭션 ID
     * 멱등성 키로 사용됨
     */
    private String traceId;
    /**
     * 요청자 ID (SSE 알림 발송 타겟)
     * Consumer 코드에서 .getRequesterId()로 사용됨
     */
    private String requesterId;

    /**
     * 결과 상태 ("SUCCESS" or "FAILED")
     * Consumer 코드에서 .getResultStatus()로 사용됨
     */
    private String resultStatus;

    // --- 아래는 성공/실패 처리를 위해 필요한 추가 정보들 ---

    /**
     * 생성된 컨테이너 ID (Docker ID)
     */
    private String containerId;

    /**
     * 생성된 컨테이너 이름 (Container Name)
     */
    private String hostContainerName;

    /**
     * 실패 시 에러 메시지
     */
    private String failureReason;

    /**
     * 포트 바인딩 정보 (ContainerPort -> HostPort)
     */
    private Map<String, Integer> portBindings;
}