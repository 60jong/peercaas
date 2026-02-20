package dev._60jong.peercaas.hub.domain.agent.controller.api.request;

import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Getter;
import lombok.NoArgsConstructor;

@Getter
@NoArgsConstructor
public class RegisterClientAgentRequest {
    private String key;
    private Long memberId;

    @JsonIgnore
    private String ipAddress;

    public void setClientAddress(String ipAddress) {
        this.ipAddress = ipAddress;
    }
}
