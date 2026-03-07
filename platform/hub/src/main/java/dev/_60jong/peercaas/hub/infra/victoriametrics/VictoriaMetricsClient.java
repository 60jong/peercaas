package dev._60jong.peercaas.hub.infra.victoriametrics;

import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.http.ResponseEntity;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestTemplate;
import org.springframework.web.util.UriComponentsBuilder;

import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.Collections;
import java.util.List;
import java.util.Map;

@Slf4j
@Component
public class VictoriaMetricsClient {

    private final RestTemplate restTemplate;
    private final String baseUrl;
    private final String authHeader;

    public VictoriaMetricsClient(
            RestTemplate restTemplate,
            @Value("${victoriametrics.url}") String url,
            @Value("${victoriametrics.user}") String user,
            @Value("${victoriametrics.password}") String password
    ) {
        this.restTemplate = restTemplate;
        this.baseUrl = url;
        this.authHeader = "Basic " + Base64.getEncoder()
                .encodeToString((user + ":" + password).getBytes(StandardCharsets.UTF_8));
    }

    /**
     * MetricsQL 쿼리를 실행해 시계열 값 목록을 반환한다.
     * 반환값: [[epochSeconds, "valueString"], ...] — VM query_range 결과의 첫 번째 series
     */
    @SuppressWarnings("unchecked")
    public List<List<Object>> queryRange(String query, long startEpoch, long endEpoch, String step) {
        // String 기반의 URL은 RestTemplate이 내부적으로 변수 확장({variable})을 시도하여
        // PromQL의 중괄호와 충돌하므로, URI 객체를 직접 생성하여 전달해야 한다.
        URI uri = UriComponentsBuilder.fromHttpUrl(baseUrl + "/api/v1/query_range")
                .queryParam("query", query)
                .queryParam("start", startEpoch)
                .queryParam("end", endEpoch)
                .queryParam("step", step)
                .build()
                .encode()
                .toUri();

        HttpHeaders headers = new HttpHeaders();
        headers.set("Authorization", authHeader);

        try {
            ResponseEntity<Map> response = restTemplate.exchange(
                    uri, HttpMethod.GET, new HttpEntity<>(headers), Map.class);
            Map<?, ?> body = response.getBody();
            if (body == null) return Collections.emptyList();

            Map<?, ?> data = (Map<?, ?>) body.get("data");
            if (data == null) return Collections.emptyList();

            List<Map<?, ?>> result = (List<Map<?, ?>>) data.get("result");
            if (result == null || result.isEmpty()) return Collections.emptyList();

            Object values = result.get(0).get("values");
            if (values == null) return Collections.emptyList();

            return (List<List<Object>>) values;
        } catch (Exception e) {
            log.warn("VictoriaMetrics query failed: query={}, error={}", query, e.getMessage());
            return Collections.emptyList();
        }
    }

    /**
     * sum(), rate() 등 집계 쿼리 결과를 위한 메서드.
     */
    public List<List<Object>> queryRangeSum(String query, long startEpoch, long endEpoch, String step) {
        return queryRange(query, startEpoch, endEpoch, step);
    }
}
