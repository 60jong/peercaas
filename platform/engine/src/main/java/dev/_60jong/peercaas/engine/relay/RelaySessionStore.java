package dev._60jong.peercaas.engine.relay;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.util.Optional;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;

@Component
public class RelaySessionStore {

    private final ConcurrentHashMap<String, RelaySession> sessions = new ConcurrentHashMap<>();

    @Value("${relay.session-ttl-seconds}")
    private long sessionTtlSeconds;

    public RelaySession create(String portKey) {
        String token = UUID.randomUUID().toString().replace("-", "");
        RelaySession session = new RelaySession(token, portKey);
        sessions.put(token, session);
        return session;
    }

    public Optional<RelaySession> get(String token) {
        return Optional.ofNullable(sessions.get(token));
    }

    public void remove(String token) {
        sessions.remove(token);
    }

    // 만료된 세션 정리 (1분마다)
    @Scheduled(fixedDelay = 60_000)
    public void evictExpired() {
        sessions.entrySet().removeIf(e -> e.getValue().isExpired(sessionTtlSeconds));
    }
}
