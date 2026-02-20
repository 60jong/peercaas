package dev._60jong.peercaas.hub.domain.agent.model;

public enum AgentStatus {
    READY, // 연결을 기다리는 상태
    CONNECTING, // 서버와 연결 중인 상태
    CONNECTED, // 서버와 연결된 상태
    DISCONNECTED, // 서버와 연결이 끊긴 상태
    ACTIVE, // 연결 완료 후 현재 활성화된 상태
    INACTIVE, // 연결 완료 후 비활성화된 상태
    ERROR // Agent 상태가 확인이 안되는 상태
}
