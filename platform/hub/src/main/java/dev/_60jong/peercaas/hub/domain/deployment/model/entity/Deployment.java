package dev._60jong.peercaas.hub.domain.deployment.model.entity;

import dev._60jong.peercaas.common.domain.model.entity.BaseTimeEntity;
import dev._60jong.peercaas.hub.domain.container.model.entity.Container;
import dev._60jong.peercaas.hub.domain.deployment.converter.MapToJsonConverter;
import dev._60jong.peercaas.hub.domain.deployment.converter.PortListToJsonConverter;
import dev._60jong.peercaas.hub.domain.deployment.model.DeploymentStatus;
import dev._60jong.peercaas.hub.domain.deployment.model.vo.CreateDeploymentRequestPayload;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import jakarta.persistence.*;
import lombok.AccessLevel;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.List;
import java.util.Map;

import static jakarta.persistence.ConstraintMode.NO_CONSTRAINT;

@Getter
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@Entity
public class Deployment extends BaseTimeEntity {

    @Id @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    private String traceId;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "requester_id", nullable = false, foreignKey = @ForeignKey(NO_CONSTRAINT))
    private Member requester;

    // --- 타겟 워커 정보 ---
    @Column(nullable = false)
    private String workerId;

    // --- 컨테이너 기본 정보 ---
    @Column(nullable = false)
    private String containerName; // 요청된 컨테이너 이름

    @Column(nullable = false)
    private String image;         // "nginx"

    @Column(nullable = false)
    private String tag;           // "latest"

    private String registry;      // "docker.io"

    // --- 상태 관리 ---
    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private DeploymentStatus status;

    private String failureReason; // 실패 시 에러 메시지 저장

    // --- 리소스 스펙 (집계를 위해 컬럼 분리) ---
    private Double cpuLimit;      // 0.5
    private Long memoryMbLimit;   // 512

    @Convert(converter = PortListToJsonConverter.class)
    @Column(columnDefinition = "json") // MySQL JSON 타입
    private List<CreateDeploymentRequestPayload.PortMapping> ports;

    @Convert(converter = MapToJsonConverter.class)
    @Column(columnDefinition = "json")
    private Map<String, String> envVars;

    private String restartPolicy;

    // --- 런타임 인스턴스 (Container 관계의 역방향) ---
    @OneToOne(mappedBy = "deployment", fetch = FetchType.LAZY)
    private Container container;

    @Builder
    public Deployment(String traceId, Member requester, String workerId, String containerName, String image, String tag,
                      String registry, Double cpuLimit, Long memoryMbLimit,
                      List<CreateDeploymentRequestPayload.PortMapping> ports,
                      Map<String, String> envVars, String restartPolicy) {
        this.traceId = traceId;

        this.requester = requester;
        this.requester.addDeployment(this);

        this.workerId = workerId;
        this.containerName = containerName;
        this.image = image;
        this.tag = tag;
        this.registry = registry;
        this.cpuLimit = cpuLimit;
        this.memoryMbLimit = memoryMbLimit;
        this.ports = ports;
        this.envVars = envVars;
        this.restartPolicy = restartPolicy;
        this.status = DeploymentStatus.PENDING; // 초기 상태
    }

    // --- 비즈니스 메서드 (상태 변경) ---
    public void updateStatus(DeploymentStatus status) {
        this.status = status;
    }

    public void markAsRunning() {
        this.status = DeploymentStatus.RUNNING;
    }

    public void updateWorkerId(String workerId) {
        this.workerId = workerId;
    }
}
