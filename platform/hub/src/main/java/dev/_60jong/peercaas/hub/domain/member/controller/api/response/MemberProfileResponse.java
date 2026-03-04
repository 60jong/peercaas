package dev._60jong.peercaas.hub.domain.member.controller.api.response;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;

@Getter
@NoArgsConstructor
@AllArgsConstructor
public class MemberProfileResponse {
    private Long id;
    private String email;
    private String nickname;
    private String clientKey;
}
