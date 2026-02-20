package dev._60jong.peercaas.hub.domain.notification.controller.response;

import lombok.AllArgsConstructor;
import lombok.Getter;

@Getter
@AllArgsConstructor
public class DeploymentEventResponse {
    private String traceId;
    private String status;    // "SUCCESS" or "FAILED"
    private String containerId; // null if FAILED
}