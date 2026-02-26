package dev._60jong.peercaas.hub.domain.agent.model.entity;

import dev._60jong.peercaas.common.domain.model.entity.BaseTimeEntity;
import dev._60jong.peercaas.hub.domain.agent.model.AgentStatus;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import jakarta.persistence.*;
import lombok.AccessLevel;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;

import static dev._60jong.peercaas.hub.domain.agent.model.AgentStatus.READY;

@Getter
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@Entity
@Table(name = "worker_agent")
public class WorkerAgent extends BaseTimeEntity {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(unique = true, nullable = false)
    private String workerId; // 워커 식별자 (예: UUID or Hostname)

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "member_id", foreignKey = @ForeignKey(ConstraintMode.NO_CONSTRAINT))
    private Member member;

    private String ipAddress;

    private Double totalCpu;
    private Double availableCpu;
    private Long totalMemoryMb;
    private Long availableMemoryMb;
    private Double averageLatencyMs;

    private LocalDateTime lastHeartbeatAt;

    @Enumerated(EnumType.STRING)
    private AgentStatus status = READY;

    @Builder
    public WorkerAgent(String workerId, Member member, String ipAddress, Double totalCpu, Long totalMemoryMb) {
        this.workerId = workerId;
        this.member = member;
        this.ipAddress = ipAddress;
        this.totalCpu = totalCpu;
        this.availableCpu = totalCpu;
        this.totalMemoryMb = totalMemoryMb;
        this.availableMemoryMb = totalMemoryMb;
        this.averageLatencyMs = 0.0;
        this.lastHeartbeatAt = LocalDateTime.now();
    }

    public void updateHeartbeat(Double availableCpu, Long availableMemoryMb, Double averageLatencyMs) {
        this.availableCpu = availableCpu;
        this.availableMemoryMb = availableMemoryMb;
        this.averageLatencyMs = averageLatencyMs;
        this.lastHeartbeatAt = LocalDateTime.now();
        this.status = AgentStatus.ACTIVE;
    }

    public void updateStatus(AgentStatus status) {
        this.status = status;
    }
}
