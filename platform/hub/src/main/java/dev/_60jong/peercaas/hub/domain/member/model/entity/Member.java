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

    private String clientKey;
    private String workerKey;

    @OneToMany(mappedBy = "requester")
    private List<Deployment> requestedDeployments = new ArrayList<>();

    @Builder
    public Member(String nickname, String email, String password) {
        this.nickname = nickname;
        this.email = email;
        this.password = password;
        this.clientKey = generateKey();
        this.workerKey = generateKey();
    }

    // == Methods == //

    private String generateKey() {
        return java.util.UUID.randomUUID().toString().replace("-", "");
    }

    public void resetClientKey() {
        this.clientKey = generateKey();
    }

    public void resetWorkerKey() {
        this.workerKey = generateKey();
    }

    /**
     * Generate deterministic worker ID based on workerKey (matches Go implementation)
     */
    public String getGeneratedWorkerId() {
        if (this.workerKey == null) return "";
        try {
            java.security.MessageDigest digest = java.security.MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(this.workerKey.getBytes(java.nio.charset.StandardCharsets.UTF_8));
            StringBuilder hexString = new StringBuilder();
            for (byte b : hash) {
                String hex = Integer.toHexString(0xff & b);
                if (hex.length() == 1) hexString.append('0');
                hexString.append(hex);
            }
            // wk- + first 12 chars of hash (total 15)
            return ("wk-" + hexString.toString()).substring(0, 15);
        } catch (Exception e) {
            return "";
        }
    }

    public void resetPassword(String password) {
        if (!StringUtils.hasText(password)) {
            throw new BaseException(ILLEGAL_ARGUMENT, "비밀번호는 빈 값일 수 없습니다.");
        }

        this.password = password;
    }

    public void updateProfile(String nickname) {
        if (StringUtils.hasText(nickname)) {
            this.nickname = nickname;
        }
    }

    public void addDeployment(Deployment deployment) {
        this.requestedDeployments.add(deployment);
    }
}
