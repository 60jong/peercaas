package dev._60jong.peercaas.hub.domain.container.model.vo;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.List;

@Getter
@NoArgsConstructor
@AllArgsConstructor
public class RelayConnectPayload {

    private String containerId;
    private String relayHost;
    private int relayPort;
    private List<SessionEntry> sessions;

    @Getter
    @NoArgsConstructor
    @AllArgsConstructor
    public static class SessionEntry {
        private String portKey;
        private String token;
    }
}
