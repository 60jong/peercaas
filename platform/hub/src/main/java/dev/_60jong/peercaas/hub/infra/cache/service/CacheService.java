package dev._60jong.peercaas.hub.infra.cache.service;

import java.util.Optional;

public interface CacheService {
    /**
     * 캐시에 데이터를 저장합니다.
     * @param cacheName 캐시 저장소 이름 (예: "users", "products")
     * @param key 데이터 식별 키
     * @param value 저장할 데이터
     */
    void put(String cacheName, String key, Object value);

    /**
     * 캐시에서 데이터를 조회합니다.
     * @param cacheName 캐시 저장소 이름
     * @param key 데이터 식별 키
     * @param type 반환할 데이터 타입 클래스
     * @return 조회된 데이터 (Optional)
     */
    <T> Optional<T> get(String cacheName, String key, Class<T> type);

    /**
     * 캐시에서 특정 데이터를 삭제합니다.
     * @param cacheName 캐시 저장소 이름
     * @param key 데이터 식별 키
     */
    void evict(String cacheName, String key);
}