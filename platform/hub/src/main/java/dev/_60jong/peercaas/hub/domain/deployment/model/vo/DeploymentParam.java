package dev._60jong.peercaas.hub.domain.deployment.model.vo;

import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;

import java.util.List;
import java.util.Map;


@Getter
@Builder
@AllArgsConstructor
public class DeploymentParam {
    private String traceId;
    private Member requester;
    private String workerId;
    private String containerName;
    private String image;
    private String tag;
    private String registry;
    private Double cpuLimit;
    private Long memoryMbLimit;
    private List<CreateDeploymentRequestPayload.PortMapping> ports;
    private Map<String, String> envVars;
    private String restartPolicy;
}
