package dev._60jong.peercaas.engine.relay.api.response;

import lombok.AllArgsConstructor;
import lombok.Getter;

import java.util.List;

@Getter
@AllArgsConstructor
public class CreateRelaySessionsResponse {

    private String relayHost;
    private int relayPort;
    private List<SessionEntry> sessions;

    @Getter
    @AllArgsConstructor
    public static class SessionEntry {
        private String portKey;
        private String token;
    }
}
