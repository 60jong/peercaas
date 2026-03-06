package dev._60jong.peercaas.hub.domain.agent.service;

import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;
import dev._60jong.peercaas.hub.domain.agent.repository.WorkerAgentRepository;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.server.ResponseStatusException;

@RequiredArgsConstructor
@Transactional
@Service
public class WorkerAgentService {

    private final WorkerAgentRepository workerAgentRepository;

    public void initialize(Member member, String workerId, String ipAddress) {
        WorkerAgent worker = workerAgentRepository.findByMemberId(member.getId())
                .orElseGet(() -> {
                    WorkerAgent newWorker = WorkerAgent.builder()
                            .workerId(workerId)
                            .member(member)
                            .ipAddress(ipAddress)
                            .build();
                    return workerAgentRepository.save(newWorker);
                });

        // 1. Check if this worker actually belongs to the member who provided the key
        if (!worker.getMember().getId().equals(member.getId())) {
            throw new ResponseStatusException(HttpStatus.FORBIDDEN, "This workerId is already registered to another user.");
        }

        // 2. Check IP mapping
        if (worker.getIpAddress() != null && !worker.getIpAddress().equals(ipAddress)) {
            throw new ResponseStatusException(HttpStatus.FORBIDDEN, 
                "This worker is already registered with a different IP: " + worker.getIpAddress() + 
                ". You must delete/reset the previous IP registration from the Hub (using --reset flag) before starting from this IP.");
        }

        // 3. If no IP is registered, register the current one
        if (worker.getIpAddress() == null) {
            worker.updateIpAddress(ipAddress);
        }
    }

    public void resetIpAddress(String workerId) {
        workerAgentRepository.findByWorkerId(workerId)
                .ifPresent(WorkerAgent::clearIpAddress);
    }
}
