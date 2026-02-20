package dev._60jong.peercaas.hub.domain.agent.service;

import dev._60jong.peercaas.hub.domain.agent.controller.api.request.RegisterClientAgentRequest;
import dev._60jong.peercaas.hub.infra.cache.service.CacheService;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import dev._60jong.peercaas.hub.domain.member.service.MemberService;
import jakarta.transaction.Transactional;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;
import org.springframework.web.server.ResponseStatusException;

import static dev._60jong.peercaas.hub.domain.agent.config.AgentConstants.CLIENT_AGENT_KEY_NAME;

@RequiredArgsConstructor
@Transactional
@Service
public class AgentFacade {

    private final ClientAgentService clientAgentService;
    private final MemberService memberService;
    private final CacheService cacheService;

    public void registerClientAgent(RegisterClientAgentRequest request) {
        String key = request.getKey();
        String ipAddr = request.getIpAddress();

        // 캐시로 부터 key에 매핑된 member-id 가져온다.
        Long memberIdMappedByKey = cacheService.get(CLIENT_AGENT_KEY_NAME, key, Long.class)
                .orElseThrow(() -> new ResponseStatusException(HttpStatus.NOT_FOUND));

        Member member = memberService.findById(memberIdMappedByKey);

        clientAgentService.create(member, ipAddr);
    }
}
