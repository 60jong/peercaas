package dev._60jong.peercaas.hub.domain.container.controller.api.response;

import lombok.AllArgsConstructor;
import lombok.Getter;

@Getter
@AllArgsConstructor
public class RelayContainerResponse {
    private String relayHost;
    private int relayPort;
    private String token;   // 이 portKey에 대한 세션 토큰
    private String portKey;
}
