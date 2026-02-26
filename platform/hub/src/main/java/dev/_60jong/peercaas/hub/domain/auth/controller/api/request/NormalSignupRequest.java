package dev._60jong.peercaas.hub.domain.auth.controller.api.request;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
public class NormalSignupRequest {
    private String nickname;
    private String email;
    private String password;
}
