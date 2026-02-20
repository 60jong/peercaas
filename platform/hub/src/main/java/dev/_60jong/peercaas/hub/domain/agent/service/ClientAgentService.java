package dev._60jong.peercaas.hub.domain.agent.service;

import dev._60jong.peercaas.hub.domain.agent.model.entity.ClientAgent;
import dev._60jong.peercaas.hub.domain.agent.repository.ClientAgentRepository;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import jakarta.transaction.Transactional;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

@Slf4j
@RequiredArgsConstructor
@Transactional
@Service
public class ClientAgentService {

    private final ClientAgentRepository clientAgentRepository;

    public void create(Member member, String ipAddr) {
        ClientAgent clientAgent = new ClientAgent(member, ipAddr);

        clientAgentRepository.save(clientAgent);
        log.info("Client agent created: {}", clientAgent);
    }
}
