package dev._60jong.peercaas.hub.domain.dashboard.controller.view;

import dev._60jong.peercaas.hub.domain.agent.repository.WorkerAgentRepository;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import dev._60jong.peercaas.hub.domain.member.service.MemberService;
import dev._60jong.peercaas.hub.global.aspect.auth.Authenticated;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;

@Controller
@RequiredArgsConstructor
@RequestMapping("/dashboard")
public class DashboardViewController {

    private final MemberService memberService;
    private final WorkerAgentRepository workerAgentRepository;

    @GetMapping
    public String dashboardMain() {
        return "dashboard/layout";
    }

    @GetMapping("/client")
    public String clientDashboard(@Authenticated Long memberId, Model model) {
        Member member = memberService.findById(memberId);
        model.addAttribute("key", member.getClientKey());
        return "dashboard/client";
    }

    @GetMapping("/worker")
    public String workerDashboard(@Authenticated Long memberId, Model model) {
        Member member = memberService.findById(memberId);
        String generatedWorkerId = member.getGeneratedWorkerId();
        
        model.addAttribute("key", member.getWorkerKey());
        model.addAttribute("workerId", generatedWorkerId);
        
        boolean isRegistered = workerAgentRepository.findByWorkerId(generatedWorkerId).isPresent();
        model.addAttribute("isWorker", isRegistered);
        
        return "dashboard/worker";
    }

    @GetMapping("/settings")
    public String settingsPage() {
        return "settings";
    }
}
