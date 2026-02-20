package dev._60jong.peercaas.hub.domain.agent.model.entity;

import dev._60jong.peercaas.common.domain.model.entity.BaseTimeEntity;
import dev._60jong.peercaas.hub.domain.agent.model.AgentStatus;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import jakarta.persistence.*;
import lombok.AccessLevel;
import lombok.Getter;
import lombok.NoArgsConstructor;

import static dev._60jong.peercaas.hub.domain.agent.model.AgentStatus.*;

@Getter
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@Entity
@Table(name = "client_agent")
public class ClientAgent extends BaseTimeEntity {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "member_id", foreignKey = @ForeignKey(ConstraintMode.NO_CONSTRAINT))
    private Member member;

    private String ipAddress;

    @Enumerated(EnumType.STRING)
    private AgentStatus status = READY;

    // Constructor //
    public ClientAgent(Member member, String ipAddress) {
        this.member = member;
        this.ipAddress = ipAddress;
    }
}
