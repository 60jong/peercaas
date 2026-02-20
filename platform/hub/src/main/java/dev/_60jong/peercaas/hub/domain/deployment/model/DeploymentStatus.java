package dev._60jong.peercaas.hub.domain.deployment.model;

public enum DeploymentStatus {
    PENDING,    // 1. DB 생성됨, MQ 전송 전
    REQUESTED,  // 2. MQ로 워커에게 전송 완료
    RUNNING,    // 3. 워커로부터 성공 응답 받음 (Async)
    STOPPED,    // 4. 중지됨
    FAILED      // 5. 실패함
}