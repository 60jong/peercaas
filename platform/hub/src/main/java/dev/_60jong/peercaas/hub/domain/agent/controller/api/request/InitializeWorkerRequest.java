package dev._60jong.peercaas.hub.domain.agent.controller.api.request;

import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

@Getter
@Setter
@NoArgsConstructor
public class InitializeWorkerRequest {
    private String workerId;
    private String workerKey;
    private String ipAddress; // From HttpServletRequest
}
