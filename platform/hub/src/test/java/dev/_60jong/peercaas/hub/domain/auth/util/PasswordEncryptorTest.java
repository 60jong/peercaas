package dev._60jong.peercaas.hub.domain.auth.util;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;

class PasswordEncryptorTest {

    private final PasswordEncryptor encryptor = new PasswordEncryptor();

    @Test
    @DisplayName("암호화된 비밀번호는 원본 비밀번호와 매칭되어야 한다 (정상 케이스)")
    void encrypt_and_match_success() {
        // given
        String rawPassword = "mySuperSecretPassword123!";

        // when
        String encodedPassword = encryptor.encrypt(rawPassword);

        // then
        // 1. 암호화된 문자열은 null이 아니어야 함
        assertThat(encodedPassword).isNotNull();
        // 2. BCrypt 해시 길이는 항상 60자여야 함 (DB 컬럼 길이 체크용)
        assertThat(encodedPassword).hasSize(60);
        // 3. 매칭 성공 확인
        assertThat(encryptor.match(rawPassword, encodedPassword)).isTrue();
    }

    @Test
    @DisplayName("틀린 비밀번호를 입력하면 매칭에 실패해야 한다")
    void match_fail_with_wrong_password() {
        // given
        String rawPassword = "password123";
        String wrongPassword = "password1234"; // 틀린 비번

        // when
        String encodedPassword = encryptor.encrypt(rawPassword);

        // then
        assertThat(encryptor.match(wrongPassword, encodedPassword)).isFalse();
    }

    @Test
    @DisplayName("같은 비밀번호로 암호화해도 매번 다른 해시값(Salt)이 생성되지만, 둘 다 매칭은 성공해야 한다")
    void salt_generation_check() {
        // given
        String rawPassword = "samePassword";

        // when
        String hash1 = encryptor.encrypt(rawPassword);
        String hash2 = encryptor.encrypt(rawPassword);

        // then
        // 1. 솔트(Salt) 때문에 해시값 문자열 자체는 달라야 함
        assertThat(hash1).isNotEqualTo(hash2);

        // 2. 하지만 둘 다 원본 비밀번호와는 매칭되어야 함
        assertThat(encryptor.match(rawPassword, hash1)).isTrue();
        assertThat(encryptor.match(rawPassword, hash2)).isTrue();
    }
}