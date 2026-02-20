package dev._60jong.peercaas.hub.domain.deployment.consumer;

import dev._60jong.peercaas.common.util.KeyGenerator;
import dev._60jong.peercaas.hub.domain.deployment.model.DeploymentStatus;
import dev._60jong.peercaas.hub.domain.deployment.model.vo.DeleteContainerPayload;
import dev._60jong.peercaas.hub.domain.deployment.model.vo.DeploymentResultPayload;
import dev._60jong.peercaas.hub.domain.deployment.service.DeploymentService;
import dev._60jong.peercaas.hub.domain.notification.controller.response.DeploymentEventResponse;
import dev._60jong.peercaas.hub.domain.notification.service.NotificationService;
import dev._60jong.peercaas.hub.global.messaging.CommandMessage;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.stereotype.Component;

import java.time.Instant;

@Slf4j
@Component
@RequiredArgsConstructor
public class DeploymentResultConsumer {

    private final DeploymentService deploymentService;
    private final NotificationService notificationService;

    private final RabbitTemplate rabbitTemplate;

    @RabbitListener(queues = "peercaas.worker.events")
    public void handleDeploymentResult(CommandMessage<DeploymentResultPayload> message) {
        String eventId = KeyGenerator.generate();
        log.info("Received deployment result for TraceID: {} / EventID: {}", message.getTraceId(), eventId);

        DeploymentResultPayload payload = message.getPayload();

        try {
            // 1. Worker가 실패했다고 응답한 경우 -> 그냥 실패 처리하고 끝 (보상 불필요)
            if (!"SUCCESS".equals(payload.getResultStatus())) {
                deploymentService.updateStatusByTraceId(message.getTraceId(), DeploymentStatus.FAILED);
                notifyClient(message.getTraceId(), eventId, "FAILED", null);
                return;
            }

            // 2. Worker는 성공함 -> 이제 Hub DB 업데이트 시도
            // [위험 구간] 여기서 DB 에러나 다른 예외가 발생하면 보상 트랜잭션 필요
            deploymentService.updateRunningInfo(message.getTraceId(), payload);
            // 3. 클라이언트 알림
            notifyClient(message.getTraceId(), eventId, "SUCCESS", payload.getContainerId());

        } catch (Exception e) {
            log.error("Error processing deployment result for TraceID: {}", message.getTraceId(), e);

            // [보상 트랜잭션 로직]
            // Worker는 성공했으나(컨테이너 만듦), Hub가 처리에 실패했으므로 Worker에게 "삭제(Undo)" 명령 발송
            if ("SUCCESS".equals(payload.getResultStatus())) {
                log.warn("Triggering Compensation Transaction (DELETE) for TraceID: {}", message.getTraceId());
                sendCompensationCommand(message.getTraceId(), payload);
            }

            // Hub 데이터는 FAILED로 일관성 맞춤
            // (트랜잭션 롤백이 되었을 테니, 새로운 트랜잭션에서 상태만 실패로 변경)
            try {
                deploymentService.updateStatusByTraceId(message.getTraceId(), DeploymentStatus.FAILED);
                notifyClient(message.getTraceId(), eventId, "FAILED", null);
            } catch (Exception ex) {
                log.error("Failed to update status to FAILED during compensation logic", ex);
            }
        }
    }

    // --- Helper Methods ---
    private void sendCompensationCommand(String traceId, DeploymentResultPayload payload) {
        // 1. 보상 명령 생성 (컨테이너 삭제)
        CommandMessage<DeleteContainerPayload> undoCommand = CommandMessage.<DeleteContainerPayload>builder()
                .traceId(traceId)
                .cmdType("DELETE_CONTAINER") // 삭제 커맨드
                .payload(new DeleteContainerPayload(payload.getContainerId())) // 생성된 컨테이너 ID
                .timestamp(Instant.now().getEpochSecond())
                .build();

        // 2. 해당 작업을 수행했던 Worker에게 직접 발송 (Routing Key = Worker ID)
        // payload에 workerId가 반드시 포함되어 있어야 함
        rabbitTemplate.convertAndSend("", payload.getWorkerId(), undoCommand);
        log.info("Compensation command sent for TraceID: {}", traceId);
    }

    private void notifyClient(String traceId, String eventId, String status, String containerId) {
        notificationService.send(
                traceId,
                eventId,
                "deployment-status",
                new DeploymentEventResponse(traceId, status, containerId)
        );
    }
}