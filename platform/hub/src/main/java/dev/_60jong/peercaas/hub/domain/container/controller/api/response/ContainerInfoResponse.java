package dev._60jong.peercaas.hub.domain.container.controller.api.response;

import dev._60jong.peercaas.hub.domain.container.model.ContainerStatus;
import dev._60jong.peercaas.hub.domain.container.model.entity.Container;
import lombok.Getter;

import java.util.Map;

@Getter
public class ContainerInfoResponse {

    private final String containerId;
    private final ContainerStatus status;
    private final String workerId;
    private final Map<String, Integer> portBindings;

    private ContainerInfoResponse(String containerId, ContainerStatus status, String workerId, Map<String, Integer> portBindings) {
        this.containerId = containerId;
        this.status = status;
        this.workerId = workerId;
        this.portBindings = portBindings;
    }

    public static ContainerInfoResponse from(Container container) {
        return new ContainerInfoResponse(
                container.getContainerId(),
                container.getStatus(),
                container.getWorkerId(),
                container.getPortBindings()
        );
    }
}
