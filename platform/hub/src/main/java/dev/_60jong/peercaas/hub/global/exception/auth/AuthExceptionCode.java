package dev._60jong.peercaas.hub.global.exception.auth;

import dev._60jong.peercaas.hub.global.exception.ExceptionCode;
import org.springframework.http.HttpStatus;

import static org.springframework.http.HttpStatus.BAD_REQUEST;
import static org.springframework.http.HttpStatus.UNAUTHORIZED;

public enum AuthExceptionCode implements ExceptionCode {
    INVALID_PASSWORD(BAD_REQUEST, "AUTH_001", "비밀번호가 일치하지 않습니다."),
    DUPLICATE_EMAIL(BAD_REQUEST, "AUTH_002", "이미 존재하는 Email입니다."),
    EXPIRED_REFRESH_TOKEN(BAD_REQUEST, "AUTH_003", "만료된 refresh token입니다."),
    ILLEGAL_ARGUMENT(BAD_REQUEST, "AUTH_004", "잘못된 파라미터입니다."),
    INVALID_TOKEN(BAD_REQUEST, "AUTH_005", "유효하지 않은 토큰입니다."),
    TOKEN_REQUIRED(UNAUTHORIZED, "AUTH_006", "인증이 필요합니다."),
    INVALID_MEMBER_INFO(BAD_REQUEST, "AUTH_007", "잘못된 회원 정보입니다.");

    private HttpStatus status;
    private String code;
    private String message;

    AuthExceptionCode(HttpStatus status, String code, String message) {
        this.status = status;
        this.code = code;
        this.message = message;
    }

    @Override
    public String getCode() {
        return this.code;
    }

    @Override
    public String getMessage() {
        return this.message;
    }

    @Override
    public HttpStatus getHttpStatus() {
        return this.status;
    }
}
