package dev._60jong.peercaas.hub.global.exception;

import org.springframework.http.HttpStatus;

public interface ExceptionCode {
    String getCode();
    String getMessage();
    HttpStatus getHttpStatus();
}
