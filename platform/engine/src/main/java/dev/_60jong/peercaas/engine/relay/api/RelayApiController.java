package dev._60jong.peercaas.engine.relay.api;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import dev._60jong.peercaas.engine.relay.RelayService;
import dev._60jong.peercaas.engine.relay.api.request.CreateRelaySessionsRequest;
import dev._60jong.peercaas.engine.relay.api.response.CreateRelaySessionsResponse;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RestController
@RequestMapping("/api/v1/relay")
public class RelayApiController {

    private final RelayService relayService;

    @PostMapping("/sessions")
    public ApiResponse<CreateRelaySessionsResponse> createSessions(
            @RequestBody CreateRelaySessionsRequest request
    ) {
        return ApiResponse.success(relayService.createSessions(request));
    }
}
