package dev._60jong.peercaas.hub.domain.metrics;

import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;

@RequiredArgsConstructor
@Controller
public class DashboardController {

    /** GET /dashboard/client/{key}  — containerId */
    @GetMapping("/dashboard/client/{key}")
    public String clientDashboard(@PathVariable String key, Model model) {
        model.addAttribute("key", key);
        return "dashboard-client";
    }

    /** GET /dashboard/worker/{key}  — workerId */
    @GetMapping("/dashboard/worker/{key}")
    public String workerDashboard(@PathVariable String key, Model model) {
        model.addAttribute("key", key);
        return "dashboard-worker";
    }
}
