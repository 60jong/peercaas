package dev._60jong.peercaas.hub.domain.container.model.vo;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;

@Getter
@NoArgsConstructor
@AllArgsConstructor
public class WebRtcConnectPayload {

    private String containerId;
    private SdpDescription offer;
    private String replyQueue; // Worker가 answer를 발행할 큐 이름

    @Getter
    @NoArgsConstructor
    @AllArgsConstructor
    public static class SdpDescription {
        private String type;
        private String sdp;
    }
}
