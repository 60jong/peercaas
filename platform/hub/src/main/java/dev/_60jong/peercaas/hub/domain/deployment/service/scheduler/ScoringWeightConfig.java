package dev._60jong.peercaas.hub.domain.deployment.service.scheduler;

import lombok.Getter;
import lombok.Setter;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Getter
@Setter
@Component
@ConfigurationProperties(prefix = "peercaas.scoring")
public class ScoringWeightConfig {

    private Map<String, Double> weights = new HashMap<>();
    private List<String> disabled = new ArrayList<>();

    public double getWeight(String key) {
        return weights.getOrDefault(key, 0.0);
    }

    public boolean isDisabled(String key) {
        return disabled.contains(key);
    }

    public static ScoringWeightConfig of(Map<String, Double> weights) {
        ScoringWeightConfig config = new ScoringWeightConfig();
        config.setWeights(new HashMap<>(weights));
        return config;
    }

    public static ScoringWeightConfig of(Map<String, Double> weights, List<String> disabled) {
        ScoringWeightConfig config = of(weights);
        config.setDisabled(new ArrayList<>(disabled));
        return config;
    }
}
