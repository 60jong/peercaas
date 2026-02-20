package dev._60jong.peercaas.hub.domain.deployment.service;

import dev._60jong.peercaas.hub.global.messaging.CommandMessage;

public interface WorkerMessageSender {
    /**
     * 특정 워커에게 명령 메시지를 전송합니다.
     * * @param targetWorkerId 수신할 워커의 ID (Routing Key 역할)
     * @param message 전송할 표준 메시지 (CommandMessage)
     */
    void send(String targetWorkerId, CommandMessage message);
}