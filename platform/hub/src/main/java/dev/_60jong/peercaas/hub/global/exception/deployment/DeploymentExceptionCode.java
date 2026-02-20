package dev._60jong.peercaas.hub.global.exception.deployment;

import dev._60jong.peercaas.hub.global.exception.ExceptionCode;
import org.springframework.http.HttpStatus;

import static org.springframework.http.HttpStatus.NOT_FOUND;

public enum DeploymentExceptionCode implements ExceptionCode {
    ENTITY_NOT_FOUND(NOT_FOUND, "DEPLOYMENT_001", "존재하지 않는 Deployment입니다.");

    private HttpStatus status;
    private String code;
    private String message;

    DeploymentExceptionCode(HttpStatus status, String code, String message) {
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
