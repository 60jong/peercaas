package dev._60jong.peercaas.hub.domain.container.repository;

import dev._60jong.peercaas.hub.domain.container.model.entity.Container;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.util.Optional;

@Repository
public interface ContainerRepository extends JpaRepository<Container, Long> {
    Optional<Container> findByContainerId(String containerId);
}
