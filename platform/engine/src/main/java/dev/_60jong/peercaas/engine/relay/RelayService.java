package dev._60jong.peercaas.engine.relay;

import dev._60jong.peercaas.engine.relay.api.request.CreateRelaySessionsRequest;
import dev._60jong.peercaas.engine.relay.api.response.CreateRelaySessionsResponse;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.stream.Collectors;

@RequiredArgsConstructor
@Service
public class RelayService {

    private final RelaySessionStore sessionStore;

    @Value("${relay.host}")
    private String relayHost;

    @Value("${relay.port}")
    private int relayPort;

    public CreateRelaySessionsResponse createSessions(CreateRelaySessionsRequest request) {
        List<CreateRelaySessionsResponse.SessionEntry> entries = request.getPortKeys().stream()
                .map(portKey -> {
                    RelaySession session = sessionStore.create(portKey);
                    return new CreateRelaySessionsResponse.SessionEntry(portKey, session.getToken());
                })
                .collect(Collectors.toList());

        return new CreateRelaySessionsResponse(relayHost, relayPort, entries);
    }
}
