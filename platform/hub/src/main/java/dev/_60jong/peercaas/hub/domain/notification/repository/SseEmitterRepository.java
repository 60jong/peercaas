package dev._60jong.peercaas.hub.domain.notification.repository;

import org.springframework.stereotype.Repository;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;

@Repository
public class SseEmitterRepository {
    // Key: userId (또는 clientId), Value: Emitter
    private final Map<String, SseEmitter> emitters = new ConcurrentHashMap<>();

    public void save(String id, SseEmitter emitter) {
        emitters.put(id, emitter);
    }

    public void deleteById(String id) {
        emitters.remove(id);
    }

    public Optional<SseEmitter> findById(String id) {
        return Optional.ofNullable(emitters.get(id));
    }
}
