package dev._60jong.peercaas.hub.domain.dashboard.controller.view;

import dev._60jong.peercaas.hub.domain.agent.model.AgentStatus;
import dev._60jong.peercaas.hub.domain.agent.repository.WorkerAgentRepository;
import dev._60jong.peercaas.hub.domain.deployment.repository.DeploymentRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.GetMapping;

import java.time.LocalDateTime;

@Controller
@RequiredArgsConstructor
public class HomeViewController {

    private final WorkerAgentRepository workerAgentRepository;
    private final DeploymentRepository deploymentRepository;

    @GetMapping("/")
    public String home(Model model) {
        // 1. 전체 등록된 워커 수
        long totalWorkers = workerAgentRepository.count();
        
        // 2. 현재 활성화된 워커 수 (최근 30초 내 하트비트)
        long activeWorkers = workerAgentRepository.findAvailableWorkers(
                AgentStatus.ACTIVE,
                LocalDateTime.now().minusSeconds(30),
                0.0, 0L).size();

        // 3. 총 배포 횟수
        long totalDeployments = deploymentRepository.count();

        // 4. 평균 지연 시간
        double avgLatency = workerAgentRepository.findAll().stream()
                .mapToDouble(w -> w.getAverageLatencyMs() != null ? w.getAverageLatencyMs() : 0.0)
                .average()
                .orElse(0.0);

        model.addAttribute("totalWorkers", totalWorkers);
        model.addAttribute("activeWorkers", activeWorkers);
        model.addAttribute("totalDeployments", totalDeployments);
        model.addAttribute("avgLatency", String.format("%.1f", avgLatency));

        return "index";
    }
}
