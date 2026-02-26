package dev._60jong.peercaas.hub.domain.dashboard.controller.view;

import dev._60jong.peercaas.hub.domain.agent.repository.WorkerAgentRepository;
import dev._60jong.peercaas.hub.domain.auth.service.AuthService;
import dev._60jong.peercaas.hub.domain.auth.controller.api.response.GetKeyResponse;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;

@Controller
@RequiredArgsConstructor
@RequestMapping("/dashboard")
public class DashboardViewController {

    private final AuthService authService;
    private final WorkerAgentRepository workerAgentRepository;

    @GetMapping
    public String dashboardMain() {
        return "dashboard/layout";
    }

    @GetMapping("/client")
    public String clientDashboard(Model model) {
        // 실제로는 현재 로그인한 사용자의 ID를 가져와야 함 (Mock: 1L)
        GetKeyResponse response = authService.issueClientKeyByMemberId(1L);
        model.addAttribute("key", response.getKey());
        return "dashboard/client";
    }

    @GetMapping("/worker")
    public String workerDashboard(Model model) {
        // 실제로는 현재 로그인한 사용자의 ID를 가져와야 함 (Mock: 1L)
        Long memberId = 1L; 
        
        workerAgentRepository.findByMemberId(memberId).ifPresentOrElse(
            worker -> {
                model.addAttribute("isWorker", true);
                model.addAttribute("key", worker.getWorkerId());
            },
            () -> {
                model.addAttribute("isWorker", false);
            }
        );
        
        return "dashboard/worker";
    }
}
