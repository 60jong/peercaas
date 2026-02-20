package dev._60jong.peercaas.hub.domain.member.model.entity;

import dev._60jong.peercaas.common.domain.model.entity.BaseTimeEntity;
import dev._60jong.peercaas.hub.domain.deployment.model.entity.Deployment;
import dev._60jong.peercaas.hub.domain.member.model.AccountType;
import dev._60jong.peercaas.hub.global.exception.BaseException;
import jakarta.persistence.*;
import lombok.AccessLevel;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;
import org.springframework.util.StringUtils;

import java.util.ArrayList;
import java.util.List;

import static dev._60jong.peercaas.hub.global.exception.auth.AuthExceptionCode.ILLEGAL_ARGUMENT;

@Getter
@Entity
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@Table(name = "member")
public class Member extends BaseTimeEntity {

    @Id @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    private String nickname;

    private String email;

    private String password;

    @Enumerated(EnumType.STRING)
    private AccountType accountType;

    @OneToMany(mappedBy = "requester")
    private List<Deployment> requestedDeployments = new ArrayList<>();

    @Builder
    public Member(String nickname, String email, String password) {
        this.nickname = nickname;
        this.email = email;
        this.password = password;
    }

    // == Methods == //

    public void resetPassword(String password) {
        if (!StringUtils.hasText(password)) {
            throw new BaseException(ILLEGAL_ARGUMENT, "비밀번호는 빈 값일 수 없습니다.");
        }

        this.password = password;
    }

    public void addDeployment(Deployment deployment) {
        this.requestedDeployments.add(deployment);
    }
}
