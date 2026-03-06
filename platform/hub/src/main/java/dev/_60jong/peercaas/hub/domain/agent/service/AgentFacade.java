package dev._60jong.peercaas.hub.domain.agent.service;

import dev._60jong.peercaas.hub.domain.agent.controller.api.request.InitializeWorkerRequest;
import dev._60jong.peercaas.hub.domain.agent.controller.api.request.RegisterClientAgentRequest;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import dev._60jong.peercaas.hub.domain.member.service.MemberService;
import jakarta.transaction.Transactional;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@RequiredArgsConstructor
@Transactional
@Service
public class AgentFacade {

    private final ClientAgentService clientAgentService;
    private final WorkerAgentService workerAgentService;
    private final MemberService memberService;

    public void registerClientAgent(RegisterClientAgentRequest request) {
        String key = request.getKey();
        String ipAddr = request.getIpAddress();

        Member member = memberService.findByClientKey(key);

        clientAgentService.create(member, ipAddr);
    }

    public void initializeWorkerAgent(InitializeWorkerRequest request) {
        String key = request.getWorkerKey();
        String workerId = request.getWorkerId();
        String ipAddr = request.getIpAddress();

        Member member = memberService.findByWorkerKey(key);

        workerAgentService.initialize(member, workerId, ipAddr);
    }

    public void resetWorkerIp(String workerKey, String workerId) {
        Member member = memberService.findByWorkerKey(workerKey);

        workerAgentService.resetIpAddress(workerId);
    }
}
