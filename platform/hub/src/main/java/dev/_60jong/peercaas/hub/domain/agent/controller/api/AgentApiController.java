package dev._60jong.peercaas.hub.domain.agent.controller.api;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import dev._60jong.peercaas.hub.domain.agent.controller.api.request.InitializeWorkerRequest;
import dev._60jong.peercaas.hub.domain.agent.controller.api.request.RegisterClientAgentRequest;
import dev._60jong.peercaas.hub.domain.agent.service.AgentFacade;
import jakarta.servlet.http.HttpServletRequest;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;

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

    @PostMapping("/worker/init")
    public ApiResponse<Void> initializeWorker(
            @RequestBody InitializeWorkerRequest request,
            HttpServletRequest httpRequest
    ) {
        request.setIpAddress(httpRequest.getRemoteAddr());
        System.out.println(request.getIpAddress());
        agentFacade.initializeWorkerAgent(request);
        return ApiResponse.success(null);
    }

    @DeleteMapping("/worker/ip")
    public ApiResponse<Void> resetWorkerIp(
            @RequestParam String workerKey,
            @RequestParam String workerId
    ) {
        agentFacade.resetWorkerIp(workerKey, workerId);
        return ApiResponse.success(null);
    }
}
