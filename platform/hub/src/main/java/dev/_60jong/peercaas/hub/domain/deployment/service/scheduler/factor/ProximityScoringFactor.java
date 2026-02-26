package dev._60jong.peercaas.hub.domain.deployment.service.scheduler.factor;

import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.ScoringContext;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.ScoringFactor;
import dev._60jong.peercaas.hub.infra.geoip.LocationResolver;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Component;

@Component
@RequiredArgsConstructor
public class ProximityScoringFactor implements ScoringFactor {

    static final double MAX_DISTANCE_KM = 20_000.0;
    static final double EARTH_RADIUS_KM = 6371.0;

    private final LocationResolver locationResolver;

    @Override
    public String key() {
        return "proximity";
    }

    @Override
    public boolean isApplicable(ScoringContext context) {
        return context.getClientIpAddress() != null && !context.getClientIpAddress().isBlank();
    }

    @Override
    public double score(WorkerAgent worker, ScoringContext context) {
        double[] clientCoords = locationResolver.locate(context.getClientIpAddress());
        double[] workerCoords = locationResolver.locate(worker.getIpAddress());

        if (clientCoords == null || workerCoords == null) {
            return 0.5;
        }

        double distance = haversine(clientCoords[0], clientCoords[1], workerCoords[0], workerCoords[1]);
        return Math.max(0.0, 1.0 - distance / MAX_DISTANCE_KM);
    }

    static double haversine(double lat1, double lon1, double lat2, double lon2) {
        double dLat = Math.toRadians(lat2 - lat1);
        double dLon = Math.toRadians(lon2 - lon1);

        double a = Math.sin(dLat / 2) * Math.sin(dLat / 2)
                + Math.cos(Math.toRadians(lat1)) * Math.cos(Math.toRadians(lat2))
                * Math.sin(dLon / 2) * Math.sin(dLon / 2);

        double c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
        return EARTH_RADIUS_KM * c;
    }
}
