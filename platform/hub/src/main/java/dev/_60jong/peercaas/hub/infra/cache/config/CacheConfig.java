package dev._60jong.peercaas.hub.infra.cache.config;

import org.springframework.cache.annotation.EnableCaching;
import org.springframework.context.annotation.Configuration;

@Configuration
@EnableCaching
public class CacheConfig {
    // 별도의 Bean 설정이 없으면 ConcurrentMapCacheManager(In-Memory)가 작동
}
