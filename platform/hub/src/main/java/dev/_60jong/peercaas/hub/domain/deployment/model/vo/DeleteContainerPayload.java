package dev._60jong.peercaas.hub.domain.deployment.model.vo;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;

@Getter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class DeleteContainerPayload {

    /**
     * 삭제할 대상 컨테이너 ID (Docker ID)
     * 예: "a1b2c3d4e5..."
     */
    private String containerId;

    /**
     * 강제 삭제 여부 (docker rm -f)
     * 보상 트랜잭션(롤백) 시에는 보통 실행 중인 컨테이너를 즉시 지워야 하므로 true로 설정하는 편입니다.
     */
    @Builder.Default
    private boolean force = true;

    // 편의상 ID만 받는 생성자 추가 (이전 코드 호환용)
    public DeleteContainerPayload(String containerId) {
        this.containerId = containerId;
        this.force = true; // 기본값 강제 삭제
    }
}