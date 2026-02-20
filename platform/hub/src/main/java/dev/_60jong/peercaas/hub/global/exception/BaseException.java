package dev._60jong.peercaas.hub.global.exception;

import lombok.Getter;
import org.springframework.http.HttpStatus;

@Getter
public class BaseException extends RuntimeException {

    private HttpStatus status;
    private String code;
    private String message;

    public BaseException(ExceptionCode code, String message) {
        super(message);
        this.status = code.getHttpStatus();
        this.code = code.getCode();
        this.message = message;
    }
}
