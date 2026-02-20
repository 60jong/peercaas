package dev._60jong.peercaas.hub.domain.container.controller.api;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import dev._60jong.peercaas.hub.domain.container.controller.api.request.ConnectContainerRequest;
import dev._60jong.peercaas.hub.domain.container.controller.api.request.RelayContainerRequest;
import dev._60jong.peercaas.hub.domain.container.controller.api.response.ConnectContainerResponse;
import dev._60jong.peercaas.hub.domain.container.controller.api.response.ContainerInfoResponse;
import dev._60jong.peercaas.hub.domain.container.controller.api.response.RelayContainerResponse;
import dev._60jong.peercaas.hub.domain.container.service.ContainerService;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;

@RequiredArgsConstructor
@RestController
@RequestMapping("/api/v1/containers")
public class ContainerApiController {

    private final ContainerService containerService;

    /**
     * Hub WebRTC API: Docker container ID로 컨테이너 정보 조회
     * 인증 없음 (Hub WebRTC 명세)
     */
    @GetMapping("/{containerId}")
    public ApiResponse<ContainerInfoResponse> getContainer(@PathVariable String containerId) {
        return ApiResponse.success(containerService.getByContainerId(containerId));
    }

    /**
     * Hub WebRTC API: SDP offer를 Worker에 전달하고 SDP answer 반환
     * Client-Agent → Hub → Worker-Agent WebRTC 시그널링
     * 인증 없음 (Hub WebRTC 명세)
     */
    @PostMapping("/{containerId}/connect")
    public ApiResponse<ConnectContainerResponse> connect(
            @PathVariable String containerId,
            @RequestBody ConnectContainerRequest request
    ) {
        return ApiResponse.success(containerService.connect(containerId, request));
    }

    /**
     * TCP Relay 세션 요청: WebRTC 실패 시 fallback
     * Engine에 세션 생성 → Worker에 RELAY_CONNECT 발행 → 접속 정보 반환
     */
    @PostMapping("/{containerId}/relay")
    public ApiResponse<RelayContainerResponse> requestRelay(
            @PathVariable String containerId,
            @RequestBody RelayContainerRequest request
    ) {
        return ApiResponse.success(containerService.requestRelay(containerId, request));
    }
}
