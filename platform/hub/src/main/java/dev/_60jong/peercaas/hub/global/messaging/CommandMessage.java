package dev._60jong.peercaas.hub.global.messaging;

import lombok.Builder;
import lombok.Getter;

import java.time.Instant;
import java.util.UUID;

@Getter
@Builder
public class CommandMessage<T> {
    /**
     * 명령어 타입 (워커의 핸들러 매핑용)
     * 예: "CREATE_CONTAINER", "STOP_CONTAINER"
     */
    private String cmdType;

    /**
     * 분산 추적 ID (Traceability)
     * 로그 추적을 위해 필수
     */
    private String traceId;

    /**
     * 실제 비즈니스 데이터
     * json으로 파싱
     */
    private T payload;

    /**
     * 메시지 생성 시간 (Unix Timestamp)
     */
    private Long timestamp;

    // --- 편의 메서드 (팩토리 메서드) ---
    public static <T> CommandMessage<T> of(String cmdType, T payload) {
        return CommandMessage.<T>builder()
                .cmdType(cmdType)
                .payload(payload)
                .traceId(UUID.randomUUID().toString())
                .timestamp(Instant.now().getEpochSecond())
                .build();
    }

    public static <T> CommandMessage <T> of(String cmdType, String traceId, T payload) {
        return CommandMessage.<T>builder()
                .cmdType(cmdType)
                .payload(payload)
                .traceId(traceId)
                .timestamp(Instant.now().getEpochSecond())
                .build();
    }
}
