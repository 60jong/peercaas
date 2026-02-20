package dev._60jong.peercaas.hub.domain.notification.service;

import dev._60jong.peercaas.hub.domain.notification.repository.SseEmitterRepository;
import dev._60jong.peercaas.hub.infra.cache.service.CacheService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;
import java.util.Optional;

import static org.springframework.util.StringUtils.hasText;

@Slf4j
@Service
@RequiredArgsConstructor
public class NotificationService {
    // SSE 타임아웃 (예: 60분)
    private static final Long DEFAULT_TIMEOUT = 60L * 1000 * 60;
    private static final String CACHE_NAME = "sse_events";

    private final CacheService cacheService;
    private final SseEmitterRepository emitterRepository;

    /**
     * 클라이언트 SSE 구독 (연결)
     * @param sessionId 클라이언트 식별자
     * @param lastEventId (Optional) 클라이언트가 마지막으로 수신한 이벤트 ID
     */
    public SseEmitter subscribe(String sessionId, String lastEventId) {
        SseEmitter emitter = new SseEmitter(DEFAULT_TIMEOUT);
        emitterRepository.save(sessionId, emitter);

        // 라이프사이클 콜백: 만료되거나 종료되면 리스트에서 제거
        emitter.onCompletion(() -> {
            log.info("[SSE End] Complete: {}", sessionId);
            emitterRepository.deleteById(sessionId);
            cacheService.evict(CACHE_NAME, sessionId);
        });
        emitter.onTimeout(() -> {
            log.info("[SSE End] Timeout: {}", sessionId);
            emitterRepository.deleteById(sessionId);
        });
        emitter.onError((e) -> {
            log.error("[SSE Error] SessionId: {}", sessionId, e);
            emitterRepository.deleteById(sessionId);
        });

        // -----------------------------------------------------------------
        // [재연결 처리] Last-Event-ID가 있다면 놓친 데이터 전송
        // -----------------------------------------------------------------
        if (hasText(lastEventId)) {
            // CacheService 직접 조회
            Map<String, Object> cachedEvents = getCachedEvents(sessionId);

            cachedEvents.entrySet().stream()
                    .filter(entry -> !entry.getKey().equals(lastEventId)) // 이미 받은 건 제외
                    .forEach(entry -> {
                        String missedEventId = entry.getKey();
                        Object missedData = entry.getValue();

                        // 재전송
                        sendToClient(emitter, sessionId, "deployment-status", missedEventId, missedData);
                        log.info("[SSE Resend] Resending missed event {} to {}", missedEventId, sessionId);
                    });
        }
        // 503 에러 방지를 위한 더미 데이터 전송 (ID 없이 전송)
        sendToClient(emitter, sessionId, "connect", null, "connected!");

        return emitter;
    }

    /**
     * 특정 유저에게 알림 전송
     * @param eventId 이 알림의 고유 ID (TraceID). 이 값을 넣어야 클라이언트가 Last-Event-ID로 활용 가능
     */
    public void send(String sessionId, String eventId, String name, Object data) {
        // 1. [Cache] 캐시에 이벤트 저장
        if (eventId != null) {
            Map<String, Object> cachedEvents = getCachedEvents(sessionId);
            cachedEvents.put(eventId, data);
            cacheService.put(CACHE_NAME, sessionId, cachedEvents);
        }

        // 2. [Realtime] 실시간 전송
        Optional<SseEmitter> optionalEmitter = emitterRepository.findById(sessionId);
        if (optionalEmitter.isPresent()) {
            sendToClient(optionalEmitter.get(), sessionId, name, eventId, data);
        } else {
            log.warn("[SSE Send Failed] Session not found: {}", sessionId);
        }
    }

    /**
     * 실제 전송 로직 (내부 헬퍼)
     */
    private void sendToClient(SseEmitter emitter, String sessionId, String name, String eventId, Object data) {
        try {
            SseEmitter.SseEventBuilder event = SseEmitter.event()
                    .name(name)
                    .data(data);

            // 이벤트 ID가 있을 때만 설정 (TraceID 등)
            if (eventId != null) {
                event.id(eventId);
            }

            emitter.send(event);
        } catch (IOException e) {
            emitterRepository.deleteById(sessionId);
            log.error("[SSE Send Error] Connection failed for {}", sessionId, e);
        }
    }

    private Map<String, Object> getCachedEvents(String sessionId) {
        // CacheService 에서 Map.class로 조회하고 없으면 빈 HashMap 반환
        return (Map<String, Object>) cacheService.get(CACHE_NAME, sessionId, Map.class)
                .orElse(new HashMap<>());
    }
}
