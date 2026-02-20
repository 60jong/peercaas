package dev._60jong.peercaas.hub.domain.auth.controller.api.request;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.Setter;

@Getter
@AllArgsConstructor
public class ResetPasswordRequest {

    @Setter
    private Long memberId;
    private String newPassword;
}
