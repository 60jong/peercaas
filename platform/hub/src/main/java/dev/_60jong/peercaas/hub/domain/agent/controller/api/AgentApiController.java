package dev._60jong.peercaas.hub.domain.agent.controller.api;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import dev._60jong.peercaas.hub.domain.agent.service.AgentFacade;
import dev._60jong.peercaas.hub.domain.agent.controller.api.request.RegisterClientAgentRequest;
import jakarta.servlet.http.HttpServletRequest;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RestController
@RequestMapping("/api/v1/agent")
public class AgentApiController {

    private final AgentFacade agentFacade;

    @PostMapping("/client/register")
    public ApiResponse<Void> registerClient(
            @RequestBody RegisterClientAgentRequest request,
            HttpServletRequest httpRequest
    ) {
        request.setClientAddress(httpRequest.getRemoteAddr());

        agentFacade.registerClientAgent(request);
        return ApiResponse.success(null);
    }
}
