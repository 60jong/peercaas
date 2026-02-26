package dev._60jong.peercaas.hub.infra.geoip;

import com.maxmind.geoip2.DatabaseReader;
import com.maxmind.geoip2.model.CityResponse;
import jakarta.annotation.PreDestroy;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.io.Resource;
import org.springframework.stereotype.Service;

import java.io.IOException;
import java.io.InputStream;
import java.net.InetAddress;

@Slf4j
@Service
public class GeoIpService implements LocationResolver {

    private final DatabaseReader reader;

    public GeoIpService(@Value("${peercaas.geoip.database-path:classpath:GeoLite2-City.mmdb}") Resource databaseResource) {
        this.reader = loadDatabase(databaseResource);
    }

    GeoIpService(DatabaseReader reader) {
        this.reader = reader;
    }

    private static DatabaseReader loadDatabase(Resource resource) {
        if (resource == null || !resource.exists()) {
            log.warn("[GeoIP] Database not found. Proximity scoring will return neutral scores.");
            return null;
        }
        try (InputStream is = resource.getInputStream()) {
            DatabaseReader r = new DatabaseReader.Builder(is).build();
            log.info("[GeoIP] Database loaded: {}", resource.getDescription());
            return r;
        } catch (IOException e) {
            log.warn("[GeoIP] Failed to load database: {}. Proximity scoring will return neutral scores.", e.getMessage());
            return null;
        }
    }

    @PreDestroy
    public void destroy() {
        if (reader != null) {
            try {
                reader.close();
            } catch (IOException e) {
                log.warn("[GeoIP] Error closing database reader: {}", e.getMessage());
            }
        }
    }

    @Override
    public double[] locate(String ipAddress) {
        if (reader == null || ipAddress == null || ipAddress.isBlank()) {
            return null;
        }

        try {
            InetAddress addr = InetAddress.getByName(ipAddress);

            if (addr.isLoopbackAddress() || addr.isSiteLocalAddress() || addr.isLinkLocalAddress()) {
                return null;
            }

            CityResponse response = reader.city(addr);
            if (response.getLocation() != null
                    && response.getLocation().getLatitude() != null
                    && response.getLocation().getLongitude() != null) {
                return new double[]{
                        response.getLocation().getLatitude(),
                        response.getLocation().getLongitude()
                };
            }
        } catch (Exception e) {
            log.debug("[GeoIP] Failed to locate IP {}: {}", ipAddress, e.getMessage());
        }

        return null;
    }
}
