package dev._60jong.peercaas.hub.domain.deployment.model.vo;

import lombok.*;

import java.util.List;
import java.util.Map;

@Getter
@Builder
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor
public class CreateDeploymentRequestPayload {
    private String registry;
    private String image;
    private String tag;

    private String name;

    private List<PortMapping> ports;
    private Map<String, String> env;
    private List<VolumeMount> volumes;
    private ResourceLimit resources;
    private String restartPolicy;

    // --- Inner Classes ---
    @Getter
    @Builder
    @NoArgsConstructor(access = AccessLevel.PROTECTED)
    @AllArgsConstructor
    public static class PortMapping {
        private Integer containerPort;
        private Integer hostPort;
        private String protocol;
    }

    @Getter
    @Builder
    @NoArgsConstructor(access = AccessLevel.PROTECTED)
    @AllArgsConstructor
    public static class ResourceLimit {
        private Long memoryMb;
        private Double cpu;
    }

    @Getter
    @Builder
    @NoArgsConstructor(access = AccessLevel.PROTECTED)
    @AllArgsConstructor
    public static class VolumeMount {
        private String hostPath;
        private String containerPath;
        private Boolean readOnly;
    }
}