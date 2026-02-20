package dev._60jong.peercaas.hub.infra.engine;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.ParameterizedTypeReference;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpMethod;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestTemplate;

import java.util.List;
import java.util.Map;

@RequiredArgsConstructor
@Component
public class EngineClient {

    private final RestTemplate restTemplate;

    @Value("${peercaas.engine.url}")
    private String engineUrl;

    /**
     * Engine에 릴레이 세션을 생성 요청한다.
     * @param containerId 컨테이너 ID
     * @param portKeys    포트 키 목록 (e.g. ["3306/tcp"])
     */
    public RelaySessionsInfo createRelaySessions(String containerId, List<String> portKeys) {
        Map<String, Object> body = Map.of(
                "containerId", containerId,
                "portKeys", portKeys
        );

        var response = restTemplate.exchange(
                engineUrl + "/api/v1/relay/sessions",
                HttpMethod.POST,
                new HttpEntity<>(body),
                new ParameterizedTypeReference<EngineApiResponse<RelaySessionsInfo>>() {}
        );

        return response.getBody().getData();
    }

    // ---- 내부 DTO ----

    @Getter
    @NoArgsConstructor
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class EngineApiResponse<T> {
        private int code;
        private T data;
    }

    @Getter
    @NoArgsConstructor
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class RelaySessionsInfo {
        private String relayHost;
        private int relayPort;
        private List<SessionEntry> sessions;
    }

    @Getter
    @NoArgsConstructor
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class SessionEntry {
        private String portKey;
        private String token;
    }
}
