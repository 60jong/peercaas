package dev._60jong.peercaas.hub.domain.dashboard.controller.view;

import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;

@Controller
public class ImageViewController {

    @GetMapping("/images")
    public String searchResults(@RequestParam(required = false) String q, Model model) {
        model.addAttribute("query", q);
        return "images";
    }

    @GetMapping("/deploy")
    public String deployMock(@RequestParam String image, Model model) {
        model.addAttribute("image", image);
        return "deploy-mock";
    }
}
