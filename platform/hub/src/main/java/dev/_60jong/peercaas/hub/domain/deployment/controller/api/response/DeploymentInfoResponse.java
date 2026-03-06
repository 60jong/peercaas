package dev._60jong.peercaas.hub.domain.deployment.controller.api.response;

import dev._60jong.peercaas.hub.domain.container.model.entity.Container;
import dev._60jong.peercaas.hub.domain.deployment.model.DeploymentStatus;
import dev._60jong.peercaas.hub.domain.deployment.model.entity.Deployment;
import lombok.Getter;

import java.util.Map;

@Getter
public class DeploymentInfoResponse {

    private final Long deploymentId;
    private final String correlationId;
    private final DeploymentStatus status;
    private final String containerId;
    private final Map<String, Integer> portBindings;

    private DeploymentInfoResponse(Long deploymentId, String correlationId, DeploymentStatus status,
                                   String containerId, Map<String, Integer> portBindings) {
        this.deploymentId = deploymentId;
        this.correlationId = correlationId;
        this.status = status;
        this.containerId = containerId;
        this.portBindings = portBindings;
    }

    public static DeploymentInfoResponse from(Deployment deployment) {
        Container container = deployment.getContainer();
        return new DeploymentInfoResponse(
                deployment.getId(),
                deployment.getCorrelationId(),
                deployment.getStatus(),
                container != null ? container.getContainerId() : null,
                container != null ? container.getPortBindings() : null
        );
    }
}
