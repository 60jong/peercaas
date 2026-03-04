package dev._60jong.peercaas.hub.domain.deployment.service;

import dev._60jong.peercaas.common.util.KeyGenerator;
import dev._60jong.peercaas.hub.domain.container.service.ContainerService;
import dev._60jong.peercaas.hub.domain.deployment.controller.api.request.CreateDeploymentRequest;
import dev._60jong.peercaas.hub.domain.deployment.controller.api.response.CreateDeploymentResponse;
import dev._60jong.peercaas.hub.domain.deployment.controller.api.response.DeploymentInfoResponse;
import dev._60jong.peercaas.hub.domain.deployment.controller.api.response.InstanceResponse;
import dev._60jong.peercaas.hub.domain.agent.model.entity.WorkerAgent;
import dev._60jong.peercaas.hub.domain.deployment.model.DeploymentStatus;
import dev._60jong.peercaas.hub.domain.deployment.model.vo.DeleteContainerPayload;
import dev._60jong.peercaas.hub.domain.deployment.model.vo.DeploymentParam;
import dev._60jong.peercaas.hub.domain.deployment.model.vo.CreateDeploymentRequestPayload;
import dev._60jong.peercaas.hub.domain.deployment.model.entity.Deployment;
import dev._60jong.peercaas.hub.domain.deployment.model.vo.DeploymentResultPayload;
import dev._60jong.peercaas.hub.domain.deployment.repository.DeploymentRepository;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.ScoringContext;
import dev._60jong.peercaas.hub.domain.deployment.service.scheduler.WorkerScheduler;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import dev._60jong.peercaas.hub.domain.member.service.MemberService;
import dev._60jong.peercaas.hub.global.exception.BaseException;
import dev._60jong.peercaas.hub.global.messaging.CommandMessage;
import jakarta.persistence.EntityNotFoundException;
import jakarta.transaction.Transactional;
import lombok.RequiredArgsConstructor;
import java.util.List;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import static dev._60jong.peercaas.hub.domain.deployment.model.DeploymentStatus.REQUESTED;
import static dev._60jong.peercaas.hub.global.exception.deployment.DeploymentExceptionCode.ENTITY_NOT_FOUND;

@Slf4j
@RequiredArgsConstructor
@Service
public class DeploymentService {

    private final MemberService memberService;
    private final ContainerService containerService;
    private final WorkerScheduler workerScheduler;

    private final WorkerMessageSender workerMessageSender;
    private final DeploymentRepository deploymentRepository;

    @Transactional
    public CreateDeploymentResponse deploy(CreateDeploymentRequest request) {
        String traceId = KeyGenerator.generate();

        // 1. 타겟 Worker Agent 고르기
        DeploymentParam tempParam = request.toEntityParam(traceId, null, null);
        ScoringContext scoringContext = ScoringContext.builder()
                .clientIpAddress(request.getClientIpAddress())
                .requiredCpu(tempParam.getCpuLimit())
                .requiredMemoryMb(tempParam.getMemoryMbLimit())
                .build();

//        WorkerAgent targetWorker = workerScheduler.selectBestWorker(scoringContext)
//                .orElseThrow(() -> new BaseException(ENTITY_NOT_FOUND, "No available workers found"));
//
//        String targetWorkerId = targetWorker.getWorkerId();
        String targetWorkerId = "test-worker";
        log.info("[Deployment Start] TraceID: {}, Worker: {}", traceId, targetWorkerId);

        // 2. API DTO -> Entity 변환 및 저장 (상태: PENDING)
        Member requester = memberService.findById(request.getRequesterId());

        Deployment deployment = createDeployment(request.toEntityParam(traceId, targetWorkerId, requester));
        deploymentRepository.save(deployment);

        // 3. Entity -> WorkerPayload 변환
        CreateDeploymentRequestPayload payload = convertToWorkerPayload(deployment);

        // 4. MQ 전송
        CommandMessage<CreateDeploymentRequestPayload> message = CommandMessage.of(
                "CREATE_CONTAINER",
                traceId,
                payload
        );
        workerMessageSender.send(targetWorkerId, message);

        // 4. 전송 성공 시 상태 업데이트
        deployment.updateStatus(REQUESTED);

        return new CreateDeploymentResponse(
                deployment.getId(),
                traceId,
                deployment.getStatus(),
                deployment.getWorkerId()
        );
    }

    private CreateDeploymentRequestPayload convertToWorkerPayload(Deployment d) {
        return CreateDeploymentRequestPayload.builder()
                .registry(d.getRegistry())
                .image(d.getImage())
                .tag(d.getTag())
                .name(d.getContainerName())
                .env(d.getEnvVars())
                .ports(d.getPorts())
                .resources(CreateDeploymentRequestPayload.ResourceLimit.builder()
                        .cpu(d.getCpuLimit())
                        .memoryMb(d.getMemoryMbLimit())
                        .build())
                .restartPolicy(d.getRestartPolicy())
                .build();
    }

    @Transactional
    public void updateStatusByTraceId(String traceId, DeploymentStatus status) {
        Deployment deployment = deploymentRepository.findByTraceId(traceId)
                .orElseThrow(() -> new EntityNotFoundException("Deployment not found with traceId: " + traceId));

        deployment.updateStatus(status);
    }

    /**
     * 배포 성공 처리: Container 레코드 생성 + Deployment 상태 RUNNING으로 변경
     */
    @Transactional
    public void updateRunningInfo(String traceId, DeploymentResultPayload result) {
        Deployment deployment = deploymentRepository.findByTraceId(traceId)
                .orElseThrow(() -> new BaseException(ENTITY_NOT_FOUND, "Deployment not found"));

        containerService.register(deployment, result);
        deployment.markAsRunning();
    }

    @Transactional
    public void deleteByTraceId(String traceId, Long requesterId) {
        Deployment deployment = deploymentRepository.findByTraceId(traceId)
                .orElseThrow(() -> new BaseException(ENTITY_NOT_FOUND, "Deployment not found"));

        if (!deployment.getRequester().getId().equals(requesterId)) {
            throw new BaseException(ENTITY_NOT_FOUND, "Deployment not found");
        }

        // 컨테이너가 실행 중이면 워커에게 삭제 명령 전송
        if (deployment.getContainer() != null) {
            CommandMessage<DeleteContainerPayload> message = CommandMessage.of(
                    "DELETE_CONTAINER",
                    traceId,
                    new DeleteContainerPayload(deployment.getContainer().getContainerId())
            );
            workerMessageSender.send(deployment.getWorkerId(), message);
        }

        deployment.updateStatus(DeploymentStatus.STOPPED);
    }

    public List<InstanceResponse> getMyDeployments(Long requesterId) {
        return deploymentRepository.findByRequester_IdOrderByCreatedAtDesc(requesterId)
                .stream()
                .map(InstanceResponse::from)
                .toList();
    }

    @Transactional
    public DeploymentInfoResponse getByTraceId(String traceId) {
        Deployment deployment = deploymentRepository.findByTraceId(traceId)
                .orElseThrow(() -> new BaseException(ENTITY_NOT_FOUND, "Deployment not found"));

        return DeploymentInfoResponse.from(deployment);
    }

    /**
     * Helper Methods
     */
    private Deployment createDeployment(DeploymentParam param) {
        return Deployment.builder()
                .traceId(param.getTraceId())
                .workerId(param.getWorkerId())
                .requester(param.getRequester())
                .containerName(param.getContainerName())
                .image(param.getImage())
                .tag(param.getTag())
                .registry(param.getRegistry())
                .ports(param.getPorts())
                .envVars(param.getEnvVars())
                .cpuLimit(param.getCpuLimit())
                .memoryMbLimit(param.getMemoryMbLimit())
                .restartPolicy(param.getRestartPolicy())
                .build();
    }
}
