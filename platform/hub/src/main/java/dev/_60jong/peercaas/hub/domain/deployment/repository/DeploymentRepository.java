package dev._60jong.peercaas.hub.domain.deployment.repository;

import dev._60jong.peercaas.hub.domain.deployment.model.entity.Deployment;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.util.Optional;

@Repository
public interface DeploymentRepository extends JpaRepository<Deployment, Long> {
    Optional<Deployment> findByTraceId(String traceId);
}
