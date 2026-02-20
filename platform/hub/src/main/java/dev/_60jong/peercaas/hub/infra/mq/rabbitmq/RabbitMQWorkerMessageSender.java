package dev._60jong.peercaas.hub.infra.mq.rabbitmq;

import dev._60jong.peercaas.hub.domain.deployment.service.WorkerMessageSender;
import dev._60jong.peercaas.hub.global.messaging.CommandMessage;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.stereotype.Component;

@Slf4j
@Component
@RequiredArgsConstructor
public class RabbitMQWorkerMessageSender implements WorkerMessageSender {

    private final RabbitTemplate rabbitTemplate;

    @Override
    public void send(String targetWorkerId, CommandMessage message) {
        log.info("[MQ Send] To: {}, Type: {}, TraceId: {}",
                targetWorkerId, message.getCmdType(), message.getTraceId());

        // RabbitMQ 전송 로직
        // Exchange: "" (Default Exchange 사용 -> RoutingKey가 Queue 이름과 매핑됨)
        // RoutingKey: targetWorkerId (워커 ID)
        try {
            rabbitTemplate.convertAndSend("", targetWorkerId, message);
        } catch (Exception e) {
            log.error("[MQ Error] Failed to send message to worker: {}", targetWorkerId, e);
            // 필요 시 커스텀 예외(BusinessException)로 감싸서 던짐
            throw new RuntimeException("Message sending failed", e);
        }
    }
}