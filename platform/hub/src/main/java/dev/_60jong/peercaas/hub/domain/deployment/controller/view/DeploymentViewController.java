package dev._60jong.peercaas.hub.domain.deployment.controller.view;

import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.GetMapping;

@Controller
public class DeploymentViewController {

    @GetMapping("/instances")
    public String instancesPage() {
        return "instances";
    }
}
