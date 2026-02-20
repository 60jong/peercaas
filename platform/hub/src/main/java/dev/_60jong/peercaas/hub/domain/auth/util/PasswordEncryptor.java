package dev._60jong.peercaas.hub.domain.auth.util;

import at.favre.lib.crypto.bcrypt.BCrypt;
import org.springframework.stereotype.Component;

@Component
public class PasswordEncryptor {

    /**
     * 비밀번호 암호화
     * cost: 12 (기본값, 높을수록 안전하지만 느림)
     */
    public String encrypt(String rawPassword) {
        return BCrypt.withDefaults().hashToString(12, rawPassword.toCharArray());
    }

    /**
     * 비밀번호 검증
     * rawPassword: 입력받은 평문
     * encodedPassword: DB에 저장된 암호화된 문자열
     */
    public boolean match(String rawPassword, String encodedPassword) {
        BCrypt.Result result = BCrypt.verifyer().verify(rawPassword.toCharArray(), encodedPassword);
        return result.verified;
    }
}
