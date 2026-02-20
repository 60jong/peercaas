package dev._60jong.peercaas.hub.test;

import lombok.RequiredArgsConstructor;
import org.springframework.cache.Cache;
import org.springframework.cache.CacheManager;
import org.springframework.cache.concurrent.ConcurrentMapCache;
import org.springframework.cache.transaction.TransactionAwareCacheDecorator;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.HashMap;
import java.util.Map;

@RestController
@RequestMapping("/api/debug/cache")
@RequiredArgsConstructor
public class TestController {

    private final CacheManager cacheManager;

    /**
     * 모든 캐시 저장소의 데이터를 조회합니다.
     * 주의: 이 기능은 ConcurrentMapCacheManager(인메모리)에서만 작동합니다.
     */
    @GetMapping
    public Map<String, Object> getAllCacheData() {
        Map<String, Object> result = new HashMap<>();

        for (String cacheName : cacheManager.getCacheNames()) {
            Cache cache = cacheManager.getCache(cacheName);

            // 1. 트랜잭션 래퍼가 있다면 벗겨내기 (안전장치)
            if (cache instanceof TransactionAwareCacheDecorator) {
                cache = ((TransactionAwareCacheDecorator) cache).getTargetCache();
            }

            // 2. ConcurrentMapCache 확인
            if (cache instanceof ConcurrentMapCache) {
                ConcurrentMapCache mapCache = (ConcurrentMapCache) cache;
                // 내부의 실제 Map 데이터를 꺼내서 결과에 담음
                result.put(cacheName, mapCache.getNativeCache());
            } else {
                result.put(cacheName, "Unknown Type: " + cache.getClass().getName());
            }
        }
        return result;
    }
}