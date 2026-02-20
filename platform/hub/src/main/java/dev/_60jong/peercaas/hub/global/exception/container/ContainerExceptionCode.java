package dev._60jong.peercaas.hub.global.exception.container;

import dev._60jong.peercaas.hub.global.exception.ExceptionCode;
import org.springframework.http.HttpStatus;

import static org.springframework.http.HttpStatus.*;

public enum ContainerExceptionCode implements ExceptionCode {
    ENTITY_NOT_FOUND(NOT_FOUND, "CONTAINER_001", "존재하지 않는 Container입니다."),
    CONTAINER_NOT_RUNNING(CONFLICT, "CONTAINER_002", "컨테이너가 실행 중이 아닙니다."),
    WORKER_TIMEOUT(GATEWAY_TIMEOUT, "CONTAINER_003", "Worker로부터 응답을 받지 못했습니다.");

    private final HttpStatus status;
    private final String code;
    private final String message;

    ContainerExceptionCode(HttpStatus status, String code, String message) {
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
