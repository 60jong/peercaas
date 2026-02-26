package dev._60jong.peercaas.hub.infra.geoip;

/**
 * IP 주소를 위도/경도로 변환하는 인터페이스.
 * 테스트 시 mock 구현체를 주입할 수 있다.
 */
public interface LocationResolver {

    /**
     * @return [latitude, longitude] 배열, 실패 시 null
     */
    double[] locate(String ipAddress);
}
