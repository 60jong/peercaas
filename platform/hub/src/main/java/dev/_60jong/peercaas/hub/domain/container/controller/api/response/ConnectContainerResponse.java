package dev._60jong.peercaas.hub.domain.container.controller.api.response;

import dev._60jong.peercaas.hub.domain.container.model.vo.WebRtcAnswerPayload;
import lombok.Getter;

@Getter
public class ConnectContainerResponse {

    private final SdpAnswer answer;

    public ConnectContainerResponse(SdpAnswer answer) {
        this.answer = answer;
    }

    public static ConnectContainerResponse from(WebRtcAnswerPayload payload) {
        WebRtcAnswerPayload.SdpDescription sdp = payload.getAnswer();
        return new ConnectContainerResponse(new SdpAnswer(sdp.getType(), sdp.getSdp()));
    }

    @Getter
    public static class SdpAnswer {
        private final String type;
        private final String sdp;

        public SdpAnswer(String type, String sdp) {
            this.type = type;
            this.sdp = sdp;
        }
    }
}
