package dev._60jong.peercaas.hub.domain.auth.controller.api.request;

import lombok.AllArgsConstructor;
import lombok.Getter;

@Getter
@AllArgsConstructor
public class ReissueRequest {
    private String refreshToken;
}
