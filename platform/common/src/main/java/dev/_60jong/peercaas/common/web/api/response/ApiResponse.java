package dev._60jong.peercaas.common.web.api.response;

import lombok.Getter;

import java.time.LocalDateTime;

@Getter
public class ApiResponse<T> {

    private int code;
    private String message;
    private T data;
    private LocalDateTime timestamp;

    private ApiResponse(int status, String message, T data) {
        this.code = status;
        this.message = message;
        this.data = data;
        this.timestamp = LocalDateTime.now();
    }

    // 1. 성공 (200 OK)
    public static <T> ApiResponse<T> success(T data) {
        return new ApiResponse<>(200, "SUCCESS", data);
    }

    // 2. 접수됨 (202 Accepted)
    public static <T> ApiResponse<T> accepted(T data) {
        return new ApiResponse<>(202, "ACCEPTED", data); // 혹은 message를 "PROCESSING" 등으로 변경 가능
    }

    // 3. 기타 상태 (유연하게 사용)
    public static <T> ApiResponse<T> of(int status, String message, T data) {
        return new ApiResponse<>(status, message, data);
    }

    // 4. Internal Server Error
    public static ApiResponse<Void> failure(Exception e) {
        return new ApiResponse<>(500, e.getMessage(), null);
    }
}