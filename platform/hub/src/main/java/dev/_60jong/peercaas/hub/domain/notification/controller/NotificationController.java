package dev._60jong.peercaas.hub.domain.notification.controller;

import dev._60jong.peercaas.hub.domain.notification.service.NotificationService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

@Slf4j
@RequiredArgsConstructor
@RestController
@RequestMapping("/api/v1/notification")
public class NotificationController {

    private final NotificationService notificationService;

    /**
     * SSE 구독 엔드포인트
     * @param sessionId 클라이언트가 생성한 고유 ID (UUID 등)
     * @param lastEventId (Optional) 재연결 시 클라이언트가 보내주는 마지막 수신 이벤트 ID
     */
    @GetMapping(value = "/subscribe/{sessionId}", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public SseEmitter subscribe(
            @PathVariable String sessionId,
            @RequestHeader(value = "Last-Event-ID", required = false, defaultValue = "") String lastEventId
    ) {

        log.info("[SSE Subscribe] SessionId: {}, Last-Event-ID: {}", sessionId, lastEventId);

        // 서비스로 헤더 값(lastEventId)도 함께 전달
        return notificationService.subscribe(sessionId, lastEventId);
    }
}