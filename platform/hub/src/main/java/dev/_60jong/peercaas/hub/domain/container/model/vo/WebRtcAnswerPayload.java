package dev._60jong.peercaas.hub.domain.container.model.vo;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;

@Getter
@NoArgsConstructor
@AllArgsConstructor
public class WebRtcAnswerPayload {

    private SdpDescription answer;

    @Getter
    @NoArgsConstructor
    @AllArgsConstructor
    public static class SdpDescription {
        private String type;
        private String sdp;
    }
}
