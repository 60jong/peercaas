# Hub WebRTC API Specification

## Overview

Hub는 Client-Agent와 Worker-Agent 사이의 WebRTC signaling을 중개한다. Client-Agent는 Hub REST API를 통해 컨테이너 정보를 조회하고, WebRTC offer/answer 교환을 수행한다.

## Endpoints

### GET /api/v1/containers/{containerId}

컨테이너 정보를 조회한다.

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| containerId | string | Docker 컨테이너 ID |

**Response 200 OK:**
```json
{
  "containerId": "abc123def456",
  "status": "RUNNING",
  "workerId": "worker-node-01",
  "portBindings": {
    "3306/tcp": 33060,
    "80/tcp": 8080
  }
}
```

**Response 404 Not Found:**
```json
{
  "error": "container not found"
}
```

**Status Values:**
- `RUNNING` — 컨테이너가 정상 실행 중
- `CREATING` — 컨테이너 생성 중
- `STOPPED` — 컨테이너 중지됨
- `FAILED` — 컨테이너 생성 실패

---

### POST /api/v1/containers/{containerId}/connect

WebRTC signaling을 수행한다. Client의 SDP offer를 받아 Worker에 전달하고, Worker의 SDP answer를 반환한다.

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| containerId | string | Docker 컨테이너 ID |

**Request Body:**
```json
{
  "offer": {
    "type": "offer",
    "sdp": "v=0\r\n..."
  }
}
```

**Response 200 OK:**
```json
{
  "answer": {
    "type": "answer",
    "sdp": "v=0\r\n..."
  }
}
```

**Response 404 Not Found:**
컨테이너가 존재하지 않거나 Worker에 등록되지 않은 경우.

**Response 504 Gateway Timeout:**
Worker로부터 30초 내에 answer를 받지 못한 경우.

---

## Hub 내부 처리 흐름

```
Client-Agent                Hub (Java)                Worker-Agent
    |                          |                          |
    |-- GET /containers/{id} ->|                          |
    |<-- 200 (containerInfo) --|                          |
    |                          |                          |
    |-- POST /connect -------->|                          |
    |                          |-- RabbitMQ: CONNECT_WEBRTC ->|
    |                          |   (offer + replyQueue)   |
    |                          |                          |
    |                          |<-- RabbitMQ: ANSWER ------|
    |                          |   (CompletableFuture)    |
    |<-- 200 (answer) --------|                          |
    |                          |                          |
    |========== WebRTC P2P DataChannel 연결 ==============|
```

1. Client-Agent가 `POST /connect`로 SDP offer를 전송
2. Hub가 RabbitMQ를 통해 해당 Worker에 `CONNECT_WEBRTC` 메시지 발행
   - Payload: `{ containerId, offer, replyQueue }`
   - `replyQueue`는 Hub가 생성한 임시 큐 (CompletableFuture 대기용)
3. Worker가 PeerConnection을 생성하고 SDP answer를 `replyQueue`에 발행
4. Hub가 CompletableFuture에서 answer를 수신 (타임아웃: 30초)
5. Hub가 Client-Agent에 answer를 HTTP 응답으로 반환
6. Client-Agent와 Worker-Agent 사이에 WebRTC DataChannel 연결 수립

## Error Codes

| HTTP Status | Description |
|-------------|-------------|
| 200 | 성공 |
| 400 | 잘못된 요청 (offer 누락 등) |
| 404 | 컨테이너를 찾을 수 없음 |
| 409 | 컨테이너가 RUNNING 상태가 아님 |
| 504 | Worker 응답 타임아웃 (30초 초과) |
| 500 | 내부 서버 오류 |
