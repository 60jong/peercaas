package dev._60jong.peercaas.hub.global.exception;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.server.ResponseStatusException;

@Slf4j
@RestControllerAdvice
public class GlobalExceptionHandler {

    /**
     * [0] ResponseStatusException 처리
     * Spring의 기본 예외 중 상태 코드를 포함한 예외를 처리합니다.
     */
    @ExceptionHandler(ResponseStatusException.class)
    protected ResponseEntity<ApiResponse<?>> handleResponseStatusException(ResponseStatusException e) {
        log.warn("[ResponseStatusException] Status: {}, Reason: {}", e.getStatusCode(), e.getReason());
        return ResponseEntity
                .status(e.getStatusCode())
                .body(ApiResponse.of(e.getStatusCode().value(), "HTTP_ERROR", e.getReason()));
    }

    /**
     * [1] BaseException 처리
     * 개발자가 의도적으로 던진 예외들을 처리합니다.
     */
    @ExceptionHandler(BaseException.class)
    protected ResponseEntity<ApiResponse<?>> handleBaseException(BaseException e) {
        log.warn("[BaseException] Code: {}, Message: {}", e.getCode(), e.getMessage());
        return ResponseEntity
                .status(e.getStatus())
                .body(ApiResponse.of(e.getStatus().value(), e.getCode(), e.getMessage()));
    }

    /**
     * [2] catch 되지 않은 나머지 예외 처리
     * 예상치 못한 런타임 에러(NullPointerException 등)를 500으로 처리합니다.
     */
    @ExceptionHandler(Exception.class)
    protected ResponseEntity<ApiResponse<?>> handleException(Exception e) {
        log.error("[Unhandled Exception]", e);
        return ResponseEntity
                .status(500)
                .body(ApiResponse.failure(e));
    }
}
