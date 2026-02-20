package dev._60jong.peercaas.engine.relay.api.request;

import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.List;

@Getter
@NoArgsConstructor
public class CreateRelaySessionsRequest {
    private List<String> portKeys;
    private String containerId;
}
