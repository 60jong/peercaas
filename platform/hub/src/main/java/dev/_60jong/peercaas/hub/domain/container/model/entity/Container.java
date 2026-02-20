package dev._60jong.peercaas.hub.domain.container.model.entity;

import dev._60jong.peercaas.common.domain.model.entity.BaseTimeEntity;
import dev._60jong.peercaas.hub.domain.container.model.ContainerStatus;
import dev._60jong.peercaas.hub.domain.deployment.converter.PortBindingMapConverter;
import dev._60jong.peercaas.hub.domain.deployment.model.entity.Deployment;
import jakarta.persistence.*;
import lombok.AccessLevel;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.Map;

import static jakarta.persistence.ConstraintMode.NO_CONSTRAINT;

@Getter
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@Entity
public class Container extends BaseTimeEntity {

    @Id @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @OneToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "deployment_id", nullable = false, unique = true, foreignKey = @ForeignKey(NO_CONSTRAINT))
    private Deployment deployment;

    @Column(nullable = false, unique = true)
    private String containerId;

    @Column(nullable = false)
    private String containerName;

    @Column(nullable = false)
    private String workerId;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private ContainerStatus status;

    @Convert(converter = PortBindingMapConverter.class)
    @Column(columnDefinition = "json")
    private Map<String, Integer> portBindings;

    @Builder
    public Container(Deployment deployment, String containerId, String containerName,
                     String workerId, Map<String, Integer> portBindings) {
        this.deployment = deployment;
        this.containerId = containerId;
        this.containerName = containerName;
        this.workerId = workerId;
        this.portBindings = portBindings;
        this.status = ContainerStatus.RUNNING;
    }
}
