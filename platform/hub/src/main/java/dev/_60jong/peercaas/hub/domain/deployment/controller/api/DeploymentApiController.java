package dev._60jong.peercaas.hub.domain.deployment.controller.api;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import dev._60jong.peercaas.hub.domain.deployment.controller.api.request.CreateDeploymentRequest;
import dev._60jong.peercaas.hub.domain.deployment.controller.api.response.CreateDeploymentResponse;
import dev._60jong.peercaas.hub.domain.deployment.controller.api.response.DeploymentInfoResponse;
import dev._60jong.peercaas.hub.domain.deployment.controller.api.response.InstanceResponse;
import java.util.List;
import dev._60jong.peercaas.hub.domain.deployment.service.DeploymentService;
import dev._60jong.peercaas.hub.global.aspect.auth.Authenticated;
import jakarta.servlet.http.HttpServletRequest;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;

@RequiredArgsConstructor
@RestController
@RequestMapping("/api/v1/deployment")
public class DeploymentApiController {

    private final DeploymentService deploymentService;

    /**
     * Deployment 생성 API
     */
    @PostMapping("")
    public ApiResponse<CreateDeploymentResponse> createDeployment(
            @Authenticated Long requesterId,
            @RequestBody CreateDeploymentRequest request,
            HttpServletRequest httpRequest
    ) {
        request.setRequesterId(requesterId);
        request.setClientIpAddress(extractClientIp(httpRequest));

        return ApiResponse.accepted(deploymentService.deploy(request));
    }

    private String extractClientIp(HttpServletRequest request) {
        String xForwardedFor = request.getHeader("X-Forwarded-For");
        if (xForwardedFor != null && !xForwardedFor.isBlank()) {
            return xForwardedFor.split(",")[0].trim();
        }
        return request.getRemoteAddr();
    }

    /**
     * 내 Deployment 목록 조회 (최신순)
     */
    @GetMapping("")
    public ApiResponse<List<InstanceResponse>> getMyDeployments(@Authenticated Long requesterId) {
        return ApiResponse.success(deploymentService.getMyDeployments(requesterId));
    }

    /**
     * Deployment 삭제 (워커에 컨테이너 삭제 명령 전송 + 상태 STOPPED)
     */
    @DeleteMapping("/{traceId}")
    public ApiResponse<Void> deleteDeployment(
            @Authenticated Long requesterId,
            @PathVariable String traceId
    ) {
        deploymentService.deleteByTraceId(traceId, requesterId);
        return ApiResponse.success(null);
    }

    /**
     * traceId로 Deployment 상태 및 Container 정보 조회
     * SSE 알림을 놓쳤을 때 클라이언트가 폴링에 사용
     */
    @GetMapping("/{traceId}")
    public ApiResponse<DeploymentInfoResponse> getDeployment(@PathVariable String traceId) {
        return ApiResponse.success(deploymentService.getByTraceId(traceId));
    }
}
