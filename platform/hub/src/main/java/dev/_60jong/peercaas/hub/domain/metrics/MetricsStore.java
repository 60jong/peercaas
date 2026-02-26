package dev._60jong.peercaas.hub.domain.metrics;

import dev._60jong.peercaas.hub.domain.metrics.dto.AgentSnapshot;
import dev._60jong.peercaas.hub.domain.metrics.dto.MetricsReport;
import org.springframework.stereotype.Component;

import java.time.Instant;
import java.time.ZoneOffset;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;

@Component
public class MetricsStore {

    private static final long STALE_THRESHOLD_MS = 15_000;
    private static final DateTimeFormatter FMT =
            DateTimeFormatter.ofPattern("HH:mm:ss").withZone(ZoneOffset.UTC);

    private record Entry(String agentType, Instant reportedAt, MetricsReport report) {}

    private final ConcurrentHashMap<String, Entry> store = new ConcurrentHashMap<>();

    public void update(MetricsReport report) {
        store.put(report.agentId(), new Entry(report.agentType(), Instant.now(), report));
    }

    /** Returns a single AgentSnapshot by key (agentId), or empty if not found. */
    public Optional<AgentSnapshot> getByKey(String key) {
        Entry entry = store.get(key);
        if (entry == null) return Optional.empty();
        return Optional.of(toSnapshot(entry));
    }

    /** Returns [clients, workers] grouped. */
    public Map<String, List<AgentSnapshot>> getAll() {
        List<AgentSnapshot> clients = new ArrayList<>();
        List<AgentSnapshot> workers = new ArrayList<>();

        store.values().forEach(entry -> {
            AgentSnapshot snap = toSnapshot(entry);
            if ("client".equals(entry.agentType())) {
                clients.add(snap);
            } else {
                workers.add(snap);
            }
        });

        clients.sort((a, b) -> a.agentId().compareTo(b.agentId()));
        workers.sort((a, b) -> a.agentId().compareTo(b.agentId()));

        return Map.of("clients", clients, "workers", workers);
    }

    private AgentSnapshot toSnapshot(Entry entry) {
        Instant now = Instant.now();
        boolean stale = now.toEpochMilli() - entry.reportedAt().toEpochMilli() > STALE_THRESHOLD_MS;
        return new AgentSnapshot(
                entry.report().agentId(),
                entry.agentType(),
                FMT.format(entry.reportedAt()),
                stale,
                entry.report().containers()
        );
    }
}
