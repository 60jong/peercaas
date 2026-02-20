package dev._60jong.peercaas.hub.global.exception;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import lombok.extern.slf4j.Slf4j;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

@Slf4j
@RestControllerAdvice
public class GlobalExceptionHandler {

    /**
     * [1] BaseException 처리
     * 개발자가 의도적으로 던진 예외들을 처리합니다.
     */
    @ExceptionHandler(BaseException.class)
    protected ApiResponse<?> handleBaseException(BaseException e) {
        log.warn("[BaseException] Code: {}, Message: {}", e.getCode(), e.getMessage());
        return ApiResponse.of(e.getStatus().value(), e.getCode(), e.getMessage());
    }

    /**
     * [2] catch 되지 않은 나머지 예외 처리
     * 예상치 못한 런타임 에러(NullPointerException 등)를 500으로 처리합니다.
     */
    @ExceptionHandler(Exception.class)
    protected ApiResponse<?> handleException(Exception e) {
        log.error("[Unhandled Exception]", e);
        return ApiResponse.failure(e);
    }
}
