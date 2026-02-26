package dev._60jong.peercaas.hub.domain.deployment.repository;

import dev._60jong.peercaas.hub.domain.deployment.model.DeploymentStatus;
import dev._60jong.peercaas.hub.domain.deployment.model.entity.Deployment;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.time.LocalDateTime;
import java.util.Optional;

@Repository
public interface DeploymentRepository extends JpaRepository<Deployment, Long> {
    Optional<Deployment> findByTraceId(String traceId);

    long countByWorkerIdAndCreatedAtAfter(String workerId, LocalDateTime since);

    long countByWorkerIdAndStatusAndCreatedAtAfter(String workerId, DeploymentStatus status, LocalDateTime since);
}
