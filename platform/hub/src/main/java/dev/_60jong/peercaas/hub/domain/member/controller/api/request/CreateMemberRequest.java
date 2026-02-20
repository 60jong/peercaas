package dev._60jong.peercaas.hub.domain.member.controller.api.request;

import lombok.AllArgsConstructor;
import lombok.Getter;

@Getter
@AllArgsConstructor
public class CreateMemberRequest {
    private String nickname;
    private String email;
    private String password;
}
