package dev._60jong.peercaas.hub.domain.auth.controller.api;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.ReissueRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.ResetPasswordRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.NormalSigninRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.NormalSignupRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.response.TokenResponse;
import dev._60jong.peercaas.hub.domain.auth.service.AuthService;
import dev._60jong.peercaas.hub.domain.auth.controller.api.response.GetKeyResponse;
import dev._60jong.peercaas.hub.global.aspect.auth.Authenticated;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;

@RequiredArgsConstructor
@RestController
@RequestMapping("/api/v1/auth")
public class AuthApiController {

    private final AuthService authService;

    /**
     * Client 등록용 Key 발급
     */
    @GetMapping("/agent/client/key")
    public ApiResponse<GetKeyResponse> getClientKey(@RequestParam Long memberId) {
        return ApiResponse.success(authService.issueClientKeyByMemberId(memberId));
    }

    /**
     * Worker 등록 용 Key 발급
     */
    @GetMapping("/agent/worker/key")
    public ApiResponse<GetKeyResponse> getWorkerKey(@RequestParam Long memberId) {
        return ApiResponse.success(authService.issueWorkerKeyByMemberId(memberId));
    }

    /**
     * 회원가입 - 일반
     */
    @PostMapping("/signup")
    public ApiResponse<TokenResponse> signup(@RequestBody NormalSignupRequest request) {
        return ApiResponse.success(authService.signup(request));
    }

    /**
     * 로그인 - 일반
     */
    @PostMapping("/signin")
    public ApiResponse<TokenResponse> signin(@RequestBody NormalSigninRequest request) {
        return ApiResponse.success(authService.signin(request));
    }

    /**
     * Access Token 재발급
     */
    @PostMapping("/reissue")
    public ApiResponse<TokenResponse> reissue(@RequestBody ReissueRequest request) {
        return ApiResponse.success(authService.reissue(request));
    }

    /**
     * 비밀번호 재설정 - 일반
     */
    @PutMapping("/reset/password")
    public ApiResponse<Void> resetPassword(
            @Authenticated Long memberId,
            @RequestBody ResetPasswordRequest request
    ) {
        request.setMemberId(memberId);

        authService.resetPassword(request);
        return ApiResponse.success(null);
    }
}
