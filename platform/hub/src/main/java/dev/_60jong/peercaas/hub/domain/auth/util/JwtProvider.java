package dev._60jong.peercaas.hub.domain.auth.util;

import io.jsonwebtoken.Claims;
import io.jsonwebtoken.JwtException;
import io.jsonwebtoken.Jwts;
import io.jsonwebtoken.SignatureAlgorithm;
import io.jsonwebtoken.security.Keys;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.security.Key;
import java.util.Date;

@Component
public class JwtProvider {

    private final Key key;
    private final long accessExpiration;
    private long refreshExpiration;

    public JwtProvider(
            @Value("${jwt.secret}") String secret,
            @Value("${jwt.access-expiration}") long accessExpiration,
            @Value("${jwt.refresh-expiration}") long refreshExpiration
    ) {
        this.key = Keys.hmacShaKeyFor(secret.getBytes());
//        this.accessExpiration = accessExpiration;
        this.accessExpiration = 1000L * 60 * 60 * 24 * 365 * 10; // 10년

        this.refreshExpiration = refreshExpiration;
    }

    /**
     * Access Token 생성
     */
    public String createAccessToken(Long memberId) {
        return createToken(memberId, accessExpiration);
    }

    /**
     * Refresh Token 생성
     */
    public String createRefreshToken(Long memberId) {
        return createToken(memberId, refreshExpiration);
    }

    /**
     * 내부 토큰 생성 로직
     */
    private String createToken(Long memberId, long expiration) {
        Date now = new Date();
        return Jwts.builder()
                .setSubject(String.valueOf(memberId))
                .claim("id", memberId)
                .setIssuedAt(now)
                .setExpiration(new Date(now.getTime() + expiration))
                .signWith(key, SignatureAlgorithm.HS256)
                .compact();
    }

    /**
     * 토큰 남은 유효 시간(ms) 반환
     */
    public long getRemainingTime(String token) {
        try {
            Date expiration = getClaims(token).getExpiration();
            long now = new Date().getTime();
            return expiration.getTime() - now;
        } catch (Exception e) {
            return 0;
        }
    }

    /**
     * 토큰에서 사용자 ID(Subject) 추출
     */
    public Long getMemberId(String token) {
        return Long.parseLong(getClaims(token).getSubject());
    }

    /**
     * 토큰 검증
     */
    public boolean validateToken(String token) {
        try {
            getClaims(token);
            return true;
        } catch (JwtException | IllegalArgumentException e) {
            return false;
        }
    }

    private Claims getClaims(String token) {
        return Jwts.parserBuilder()
                .setSigningKey(key)
                .build()
                .parseClaimsJws(token)
                .getBody();
    }
}