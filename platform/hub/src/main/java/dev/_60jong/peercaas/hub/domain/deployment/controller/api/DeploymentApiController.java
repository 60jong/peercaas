package dev._60jong.peercaas.hub.domain.deployment.controller.api;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import dev._60jong.peercaas.hub.domain.deployment.controller.api.request.CreateDeploymentRequest;
import dev._60jong.peercaas.hub.domain.deployment.controller.api.response.CreateDeploymentResponse;
import dev._60jong.peercaas.hub.domain.deployment.controller.api.response.DeploymentInfoResponse;
import dev._60jong.peercaas.hub.domain.deployment.service.DeploymentService;
import dev._60jong.peercaas.hub.global.aspect.auth.Authenticated;
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
            @RequestBody CreateDeploymentRequest request
    ) {
        request.setRequesterId(requesterId);

        return ApiResponse.accepted(deploymentService.deploy(request));
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
