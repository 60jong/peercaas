package dev._60jong.peercaas.hub.domain.deployment.controller.api.request;

import dev._60jong.peercaas.hub.domain.deployment.model.vo.DeploymentParam;
import dev._60jong.peercaas.hub.domain.deployment.model.vo.CreateDeploymentRequestPayload.PortMapping;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import lombok.AllArgsConstructor;
import lombok.Getter;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

@Getter
public class CreateDeploymentRequest {
    // 요청자
    private Long requesterId;

    // 클라이언트 IP (서버에서 설정)
    private String clientIpAddress;

    // 컨테이너 스펙
    private String name;
    private String image;
    private String registry;

    private List<PortSpec> ports;
    private Map<String, String> env;
    private ResourceSpec resources;
    private String restartPolicy;

    @Getter
    @AllArgsConstructor
    public static class PortSpec {
        private Integer containerPort;
        private Integer hostPort;
        private String protocol; // "tcp"
    }

    @Getter
    @AllArgsConstructor
    public static class ResourceSpec {
        private Long memoryMb;
        private Double cpu;
    }

    public DeploymentParam toEntityParam(String traceId, String workerId, Member requester) {
        return DeploymentParam.builder()
                .traceId(traceId)
                .workerId(workerId)
                .requester(requester)
                .containerName(this.name)
                .image(parseImageName())      // 로직 분리
                .tag(parseTag())              // 로직 분리
                .registry(resolveRegistry())  // 기본값 처리
                .ports(convertPorts())        // 변환 로직 분리
                .envVars(resolveEnv())        // Null 처리
                .cpuLimit(resolveCpu())
                .memoryMbLimit(resolveMemory())
                .restartPolicy(this.restartPolicy)
                .build();
    }

    public void setRequesterId(Long requesterId) {
        this.requesterId = requesterId;
    }

    public void setClientIpAddress(String clientIpAddress) {
        this.clientIpAddress = clientIpAddress;
    }

    // --- 아래는 밖에서 볼 필요 없는 Private Helper 메서드들 ---

    private String parseImageName() {
        if (this.image != null && this.image.contains(":")) {
            return this.image.split(":")[0];
        }
        return this.image; // 태그가 없으면 전체가 이름
    }

    private String parseTag() {
        if (this.image != null && this.image.contains(":")) {
            return this.image.split(":")[1];
        }
        return "latest"; // 태그가 없으면 기본값 latest
    }

    private String resolveRegistry() {
        return (this.registry != null) ? this.registry : "docker.io";
    }

    private List<PortMapping> convertPorts() {
        if (this.ports == null || this.ports.isEmpty()) {
            return new ArrayList<>();
        }
        // PortSpec -> WorkerContainerPayload.PortMapping 변환
        return this.ports.stream()
                .map(p -> PortMapping.builder()
                        .containerPort(p.getContainerPort())
                        .hostPort(p.getHostPort())
                        .protocol(p.getProtocol() != null ? p.getProtocol() : "tcp")
                        .build())
                .collect(Collectors.toList());
    }

    private Map<String, String> resolveEnv() {
        return (this.env != null) ? this.env : Collections.emptyMap();
    }

    private Double resolveCpu() {
        return (this.resources != null && this.resources.getCpu() != null)
                ? this.resources.getCpu() : 0.0;
    }

    private Long resolveMemory() {
        return (this.resources != null && this.resources.getMemoryMb() != null)
                ? this.resources.getMemoryMb() : 0L;
    }
}
