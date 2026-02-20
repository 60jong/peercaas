package dev._60jong.peercaas.hub.domain.agent.repository;

import dev._60jong.peercaas.hub.domain.agent.model.entity.ClientAgent;
import org.springframework.data.jpa.repository.JpaRepository;

public interface ClientAgentRepository extends JpaRepository<ClientAgent, Long> {
}
