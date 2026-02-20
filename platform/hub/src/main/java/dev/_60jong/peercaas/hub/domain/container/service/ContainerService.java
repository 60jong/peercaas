package dev._60jong.peercaas.hub.domain.container.service;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import dev._60jong.peercaas.hub.domain.container.controller.api.request.ConnectContainerRequest;
import dev._60jong.peercaas.hub.domain.container.controller.api.request.RelayContainerRequest;
import dev._60jong.peercaas.hub.domain.container.controller.api.response.ConnectContainerResponse;
import dev._60jong.peercaas.hub.domain.container.controller.api.response.ContainerInfoResponse;
import dev._60jong.peercaas.hub.domain.container.controller.api.response.RelayContainerResponse;
import dev._60jong.peercaas.hub.domain.container.model.ContainerStatus;
import dev._60jong.peercaas.hub.domain.container.model.entity.Container;
import dev._60jong.peercaas.hub.domain.container.model.vo.RelayConnectPayload;
import dev._60jong.peercaas.hub.domain.container.model.vo.WebRtcConnectPayload;
import dev._60jong.peercaas.hub.domain.container.repository.ContainerRepository;
import dev._60jong.peercaas.hub.domain.deployment.model.entity.Deployment;
import dev._60jong.peercaas.hub.domain.deployment.model.vo.DeploymentResultPayload;
import dev._60jong.peercaas.hub.global.exception.BaseException;
import dev._60jong.peercaas.hub.global.messaging.CommandMessage;
import dev._60jong.peercaas.hub.infra.engine.EngineClient;
import jakarta.transaction.Transactional;
import lombok.RequiredArgsConstructor;
import org.springframework.amqp.core.AmqpAdmin;
import org.springframework.amqp.core.Message;
import org.springframework.amqp.core.Queue;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.UUID;

import static dev._60jong.peercaas.hub.global.exception.container.ContainerExceptionCode.*;

@RequiredArgsConstructor
@Service
public class ContainerService {

    private final ContainerRepository containerRepository;
    private final RabbitTemplate rabbitTemplate;
    private final AmqpAdmin amqpAdmin;
    private final ObjectMapper objectMapper;
    private final EngineClient engineClient;

    @Transactional
    public Container register(Deployment deployment, DeploymentResultPayload result) {
        Container container = Container.builder()
                .deployment(deployment)
                .containerId(result.getContainerId())
                .containerName(result.getHostContainerName())
                .workerId(result.getWorkerId())
                .portBindings(result.getPortBindings())
                .build();

        return containerRepository.save(container);
    }

    public ContainerInfoResponse getByContainerId(String containerId) {
        Container container = containerRepository.findByContainerId(containerId)
                .orElseThrow(() -> new BaseException(ENTITY_NOT_FOUND, "Container not found: " + containerId));

        return ContainerInfoResponse.from(container);
    }

    public ConnectContainerResponse connect(String containerId, ConnectContainerRequest request) {
        Container container = containerRepository.findByContainerId(containerId)
                .orElseThrow(() -> new BaseException(ENTITY_NOT_FOUND, "Container not found: " + containerId));

        if (container.getStatus() != ContainerStatus.RUNNING) {
            throw new BaseException(CONTAINER_NOT_RUNNING, "Container is not running: " + containerId);
        }

        // 1. 임시 Reply Queue 생성 (non-durable, auto-delete)
        String replyQueueName = "webrtc.reply." + UUID.randomUUID();
        amqpAdmin.declareQueue(new Queue(replyQueueName, false, false, true));

        try {
            // 2. Worker에 CONNECT_WEBRTC 전송 (replyQueue 포함)
            WebRtcConnectPayload connectPayload = new WebRtcConnectPayload(
                    containerId,
                    new WebRtcConnectPayload.SdpDescription(
                            request.getOffer().getType(),
                            request.getOffer().getSdp()
                    ),
                    replyQueueName
            );

            CommandMessage<WebRtcConnectPayload> command = CommandMessage.of("CONNECT_WEBRTC", connectPayload);
            rabbitTemplate.convertAndSend("", container.getWorkerId(), command);

            // 3. Worker의 answer 대기 (30초)
            Message replyMessage = rabbitTemplate.receive(replyQueueName, 30_000);
            if (replyMessage == null) {
                throw new BaseException(WORKER_TIMEOUT, "Worker did not respond within timeout");
            }

            // 4. Worker 응답 파싱
            // Worker가 보내는 구조: { cmdType, traceId, payload: { containerId, answer: { type, sdp } } }
            JsonNode root = objectMapper.readTree(replyMessage.getBody());
            JsonNode answer = root.path("payload").path("answer");

            return new ConnectContainerResponse(
                    new ConnectContainerResponse.SdpAnswer(
                            answer.path("type").asText(),
                            answer.path("sdp").asText()
                    )
            );

        } catch (BaseException e) {
            throw e;
        } catch (Exception e) {
            throw new RuntimeException("WebRTC signaling failed", e);
        } finally {
            amqpAdmin.deleteQueue(replyQueueName);
        }
    }

    /**
     * TCP Relay 세션 생성 및 Worker에 RELAY_CONNECT 커맨드 발행.
     *
     * 흐름:
     * 1. Engine에 릴레이 세션 생성 요청 (portKey 1개)
     * 2. Worker에 RELAY_CONNECT 커맨드 발행
     * 3. Client에게 relay 접속 정보 반환
     */
    public RelayContainerResponse requestRelay(String containerId, RelayContainerRequest request) {
        Container container = containerRepository.findByContainerId(containerId)
                .orElseThrow(() -> new BaseException(ENTITY_NOT_FOUND, "Container not found: " + containerId));

        if (container.getStatus() != ContainerStatus.RUNNING) {
            throw new BaseException(CONTAINER_NOT_RUNNING, "Container is not running: " + containerId);
        }

        // 1. Engine에 릴레이 세션 생성
        EngineClient.RelaySessionsInfo sessionInfo = engineClient.createRelaySessions(
                containerId,
                List.of(request.getPortKey())
        );

        EngineClient.SessionEntry entry = sessionInfo.getSessions().get(0);

        // 2. Worker에 RELAY_CONNECT 커맨드 발행
        RelayConnectPayload payload = new RelayConnectPayload(
                containerId,
                sessionInfo.getRelayHost(),
                sessionInfo.getRelayPort(),
                List.of(new RelayConnectPayload.SessionEntry(entry.getPortKey(), entry.getToken()))
        );

        CommandMessage<RelayConnectPayload> command = CommandMessage.of("RELAY_CONNECT", payload);
        rabbitTemplate.convertAndSend("", container.getWorkerId(), command);

        // 3. Client에게 접속 정보 반환
        return new RelayContainerResponse(
                sessionInfo.getRelayHost(),
                sessionInfo.getRelayPort(),
                entry.getToken(),
                entry.getPortKey()
        );
    }
}
