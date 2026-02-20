# PeerCaaS — Peer Container as a Service

원격 Docker 컨테이너의 포트를 로컬에서 직접 사용할 수 있게 해주는 WebRTC 기반 터널링 서비스.

클라이언트는 별도의 포트 포워딩이나 VPN 없이, **로컬 포트에 접속하는 것만으로** 원격 컨테이너에 연결된다.

---

## 어떻게 동작하나

```
[사용자]
   │  mysql -h 127.0.0.1 -P 3306
   ▼
[Client Agent]  ←── WebRTC DataChannel (P2P) ───→  [Worker Agent]
  로컬 포트                                              │
  리스닝                                           컨테이너 포트
                                                   브릿지
                                                        │
                                                  [Docker Container]
                                                   MySQL :3306
```

WebRTC 연결이 불가한 환경(방화벽, 대칭 NAT)에서는 Engine의 TCP Relay를 통해 자동으로 fallback하며, 백그라운드에서 WebRTC 재연결을 주기적으로 시도한다.

---

## 시스템 아키텍처

```
┌─────────────────────────────────────────────────────────┐
│                      Client 측                          │
│                                                         │
│   [사용자 앱]  ──TCP──▶  [Client Agent (Go)]            │
│                            │       ▲                    │
│                            │ WebRTC│ or TCP Relay       │
└────────────────────────────┼───────┼────────────────────┘
                             │       │
                    ┌────────▼───────┴────────┐
                    │      platform/hub        │  ← REST / SSE
                    │    (Spring Boot :8080)   │  ← RabbitMQ
                    │                         │
                    │  - 배포 관리             │
                    │  - WebRTC 시그널링       │
                    │  - Relay 세션 조율       │
                    └──────┬──────────────────┘
                           │
              ┌────────────┴────────────┐
              │                         │
   ┌──────────▼──────────┐   ┌──────────▼──────────┐
   │   platform/engine    │   │   RabbitMQ           │
   │  (Spring Boot :8090) │   │                      │
   │                      │   │  - worker queue      │
   │  - TCP Relay 서버    │   │  - reply queue       │
   │    (:6006)           │   └──────────┬───────────┘
   └──────────────────────┘              │
                                ┌────────▼────────────┐
                                │  Worker Agent (Go)   │
                                │                      │
                                │  - Docker 컨테이너   │
                                │    생성/삭제         │
                                │  - WebRTC 응답       │
                                │  - Relay 연결        │
                                └──────────────────────┘
```

---

## 컴포넌트

### platform/hub (Spring Boot)
전체 시스템의 컨트롤 플레인.

| 역할 | 설명 |
|------|------|
| 배포 관리 | 컨테이너 생성/삭제 요청을 Worker에 RabbitMQ로 전달 |
| Container 도메인 | 실행 중인 컨테이너 정보 관리 (containerId, portBindings, workerId) |
| WebRTC 시그널링 | Client의 SDP offer를 Worker에 중계하고 answer를 반환 |
| Relay 조율 | Engine에 relay 세션 생성 요청 후 Worker에 RELAY_CONNECT 전달 |
| SSE 알림 | 배포 완료/실패를 클라이언트에 실시간 푸시 |

### platform/engine (Spring Boot)
네트워크 relay 서비스.

| 역할 | 설명 |
|------|------|
| TCP Relay 서버 | 세션 토큰 기반 1:1 TCP 브릿지 (`:6006`) |
| 세션 관리 | CountDownLatch 기반 두 소켓 랑데부 |

### agents/worker (Go)
Worker 노드에서 실행되는 에이전트.

| 역할 | 설명 |
|------|------|
| 컨테이너 생성 | Docker 이미지 풀 → 컨테이너 생성/시작 → 결과 응답 |
| WebRTC 처리 | SDP answer 생성, DataChannel로 컨테이너 포트 브릿지 |
| Relay 처리 | Engine relay 서버에 연결, 컨테이너 포트와 브릿지 |

### agents/client (Go)
클라이언트 측에서 실행되는 에이전트.

| 역할 | 설명 |
|------|------|
| 로컬 포트 리스닝 | 컨테이너 포트와 동일한 번호로 로컬 TCP 리스닝 |
| ConnectionManager | WebRTC / Relay 전환 + 백그라운드 retry 전략 |

---

## 연결 전략 (ConnectionManager)

```
시작
  │
  ▼
WebRTC 연결 시도 (15초 타임아웃)
  │
  ├─ 성공 ──▶ WebRTC로 서비스
  │              │
  │              └─ 연결 끊김 ──▶ Relay 전환 + retry 루프
  │
  └─ 실패 ──▶ TCP Relay 전환
                │
                └─ 백그라운드 retry: 30s → 1m → 2m → 2m ...
                      │
                      ├─ 성공 ──▶ WebRTC hot-swap
                      │          (새 연결부터 WebRTC 사용)
                      └─ 실패 ──▶ relay 유지, 다음 retry 예약
```

---

## 전체 흐름

### 1. 컨테이너 배포

```
Client          Hub             RabbitMQ        Worker
  │                │                │              │
  ├─ POST /deployment ──────────────▶              │
  │                │                │              │
  │                ├─ CREATE_CONTAINER ────────────▶
  │                │                │              │
  │                │                │◀─ DEPLOYMENT_RESULT
  │                │                │              │
  │◀── SSE: containerId ────────────┤              │
```

### 2. WebRTC 터널 연결

```
Client Agent    Hub             RabbitMQ        Worker Agent
  │                │                │              │
  ├─ GET /containers/{id} ─────────▶              │
  │◀── containerInfo ───────────────┤              │
  │                │                │              │
  ├─ POST /containers/{id}/connect ─▶              │
  │                ├─ CONNECT_WEBRTC ──────────────▶
  │                │◀── answer (reply queue) ───────┤
  │◀── SDP answer ─┤                │              │
  │                │                │              │
  │════════════ WebRTC DataChannel (P2P) ══════════│
  │                │                │              │
  ├─ TCP :3306 ──▶ DataChannel ─────────────────▶ ├─▶ Docker :33060
```

### 3. Relay Fallback

```
Client Agent    Hub             Engine          Worker Agent
  │                │                │              │
  ├─ POST /containers/{id}/relay ───▶              │
  │                ├─ POST /relay/sessions ───────▶│ (세션 토큰 발급)
  │                ├─ RELAY_CONNECT ──────────────────────────▶
  │◀── {relayHost, token} ──────────┤              │              │
  │                │                │              │              │
  ├─ TCP connect + token ──────────▶│◀─ TCP connect + token ─────┤
  │                │                │              │
  │════════════════════ TCP Relay Bridge ══════════│
```

---

## 기술 스택

| 구분 | 기술 |
|------|------|
| Hub / Engine | Java 21, Spring Boot 3, JPA, RabbitMQ, SSE |
| Agents | Go, Pion WebRTC, Docker SDK |
| 데이터베이스 | MariaDB |
| 메시지 브로커 | RabbitMQ |
| 컨테이너 런타임 | Docker |

---

## 로컬 실행

### 사전 준비

```bash
# RabbitMQ
docker run -d --name rabbitmq \
  -e RABBITMQ_DEFAULT_USER=root \
  -e RABBITMQ_DEFAULT_PASS=991911 \
  -p 5672:5672 -p 15672:15672 \
  rabbitmq:3-management

# MariaDB
docker run -d --name mariadb \
  -e MYSQL_ROOT_PASSWORD=991911 \
  -e MYSQL_DATABASE=peercaas \
  -p 3306:3306 \
  mariadb:10.11
```

### 서비스 시작 순서

```bash
# 1. Hub (port 8080)
cd platform && ./gradlew :hub:bootRun

# 2. Engine (port 8090, relay port 6006)
cd platform && ./gradlew :engine:bootRun

# 3. Worker Agent
cd agents
WORKER_ID=worker-node-01 go run ./cmd/worker/main.go

# 4. Client Agent (컨테이너 배포 완료 후)
cd agents
CONTAINER_ID=<containerId> go run ./cmd/client/main.go
```

### 컨테이너 배포 예시

```bash
# 회원 가입 & 로그인
curl -X POST http://localhost:8080/api/v1/members \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"password123","name":"test"}'

TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/signin \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"password123"}' \
  | jq -r '.data.accessToken')

# MySQL 컨테이너 배포
curl -X POST http://localhost:8080/api/v1/deployment \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-mysql",
    "image": "mysql:8",
    "ports": [{"containerPort": 3306, "hostPort": 33060, "protocol": "tcp"}],
    "env": {"MYSQL_ROOT_PASSWORD": "secret"},
    "resources": {"memoryMb": 512, "cpu": 0.5},
    "restartPolicy": "no"
  }'

# SSE로 배포 완료 대기
curl -N "http://localhost:8080/api/v1/notifications/subscribe?clientId=client-1"
```

```bash
# Client Agent 실행 (SSE에서 받은 containerId 사용)
CONTAINER_ID=<containerId> go run ./cmd/client/main.go

# 로컬에서 MySQL 접속
mysql -h 127.0.0.1 -P 3306 -u root -psecret
```

---

## 설정

### Worker Agent (`agents/configs/worker.yaml`)

```yaml
worker:
  worker_id: worker-node-01    # Worker 식별 ID (RabbitMQ 큐 이름)
  result_queue: peercaas.worker.events
  concurrency: 5
```

### Client Agent (환경변수)

| 변수 | 필수 | 기본값 | 설명 |
|------|------|--------|------|
| `CONTAINER_ID` | ✓ | - | 접속할 컨테이너 ID |
| `HUB_URL` | | `http://localhost:8080` | Hub 서버 주소 |

---

## 프로젝트 구조

```
peercaas/
├── platform/               # JVM 서비스 (Gradle 멀티모듈)
│   ├── common/             # 공통 모듈 (ApiResponse, BaseEntity)
│   ├── hub/                # 컨트롤 플레인 서비스
│   └── engine/             # TCP Relay 서비스
│
└── agents/                 # Go 에이전트
    ├── cmd/
    │   ├── worker/         # Worker Agent 진입점
    │   └── client/         # Client Agent 진입점
    └── internal/
        ├── app/
        │   ├── worker/     # CREATE_CONTAINER, CONNECT_WEBRTC, RELAY_CONNECT 핸들러
        │   └── client/     # ConnectionManager, Tunnel, HubClient
        ├── config/
        ├── core/           # Broker, CommandHandler 인터페이스
        └── infra/
            └── mq/         # RabbitMQ 구현체
```
