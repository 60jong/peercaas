package dev._60jong.peercaas.hub.domain.deployment.controller.api.response;

import dev._60jong.peercaas.hub.domain.container.model.entity.Container;
import dev._60jong.peercaas.hub.domain.deployment.model.entity.Deployment;
import lombok.Getter;

import java.time.format.DateTimeFormatter;
import java.util.Map;

@Getter
public class InstanceResponse {
    private final Long deploymentId;
    private final String traceId;
    private final String image;          // "nginx:latest"
    private final String containerName;
    private final String status;
    private final String containerId;
    private final Map<String, Integer> portBindings;
    private final String workerId;
    private final String createdAt;

    private static final DateTimeFormatter FMT = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss");

    private InstanceResponse(Long deploymentId, String traceId, String image, String containerName,
                              String status, String containerId, Map<String, Integer> portBindings,
                              String workerId, String createdAt) {
        this.deploymentId = deploymentId;
        this.traceId = traceId;
        this.image = image;
        this.containerName = containerName;
        this.status = status;
        this.containerId = containerId;
        this.portBindings = portBindings;
        this.workerId = workerId;
        this.createdAt = createdAt;
    }

    public static InstanceResponse from(Deployment d) {
        Container container = d.getContainer();
        return new InstanceResponse(
                d.getId(),
                d.getTraceId(),
                d.getImage() + ":" + d.getTag(),
                d.getContainerName(),
                d.getStatus().name(),
                container != null ? container.getContainerId() : null,
                container != null ? container.getPortBindings() : null,
                d.getWorkerId(),
                d.getCreatedAt() != null ? d.getCreatedAt().format(FMT) : null
        );
    }
}
