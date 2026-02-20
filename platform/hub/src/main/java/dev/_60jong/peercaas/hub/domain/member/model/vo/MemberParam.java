package dev._60jong.peercaas.hub.domain.member.model.vo;

import dev._60jong.peercaas.hub.domain.member.model.AccountType;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;

@Getter
@Builder
@AllArgsConstructor
public class MemberParam {
    private String nickname;
    private String email;
    private String password;
    private AccountType accountType;
}
