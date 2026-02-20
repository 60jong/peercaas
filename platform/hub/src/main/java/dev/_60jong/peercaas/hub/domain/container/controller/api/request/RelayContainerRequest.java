package dev._60jong.peercaas.hub.domain.container.controller.api.request;

import lombok.Getter;
import lombok.NoArgsConstructor;

@Getter
@NoArgsConstructor
public class RelayContainerRequest {
    private String portKey; // e.g. "3306/tcp"
}
