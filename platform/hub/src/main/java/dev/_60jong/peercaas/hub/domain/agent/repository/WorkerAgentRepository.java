package dev._60jong.peercaas.hub.domain.agent.repository;

import dev._60jong.peercaas.hub.domain.agent.model.AgentStatus;
import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;

public interface WorkerAgentRepository extends JpaRepository<WorkerAgent, Long> {

    Optional<WorkerAgent> findByWorkerId(String workerId);

    Optional<WorkerAgent> findByMemberId(Long memberId);

    @Query("SELECT w FROM WorkerAgent w " +
           "WHERE w.status = :status " +
           "AND w.lastHeartbeatAt > :heartbeatThreshold " +
           "AND w.availableCpu >= :requiredCpu " +
           "AND w.availableMemoryMb >= :requiredMemory")
    List<WorkerAgent> findAvailableWorkers(
            @Param("status") AgentStatus status,
            @Param("heartbeatThreshold") LocalDateTime heartbeatThreshold,
            @Param("requiredCpu") Double requiredCpu,
            @Param("requiredMemory") Long requiredMemory
    );

}
