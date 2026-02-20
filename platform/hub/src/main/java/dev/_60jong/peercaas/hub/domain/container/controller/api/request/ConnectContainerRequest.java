package dev._60jong.peercaas.hub.domain.container.controller.api.request;

import lombok.Getter;
import lombok.NoArgsConstructor;

@Getter
@NoArgsConstructor
public class ConnectContainerRequest {

    private SdpOffer offer;

    @Getter
    @NoArgsConstructor
    public static class SdpOffer {
        private String type;
        private String sdp;
    }
}
