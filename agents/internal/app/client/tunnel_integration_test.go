package client_test

// TestWebRTCTunnel_DataTransfer: Hub, RabbitMQ, Docker, Engine을 모두 띄우지 않고
// 인-프로세스에서 Client ↔ Worker WebRTC 터널 전체를 검증하는 통합 테스트.
//
// 대체하는 실제 컴포넌트:
//   - Hub          → httptest.Server (FakeHub): 컨테이너 정보 반환 + WebRTC 시그널링 중계
//   - RabbitMQ     → InMemoryBroker: Worker 커맨드 큐와 reply 큐를 인-메모리로 구현
//   - Docker 컨테이너 → net.Listener TCP 에코 서버: 포트에 바인딩된 서비스 시뮬레이션
//   - Worker 프로세스 → goroutine으로 WorkerAgent + ConnectWebRTCHandler 실행

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agents/internal/app/client"
	"agents/internal/app/worker"
	"agents/internal/core"
	"agents/internal/metrics"
	"agents/internal/testutil"

	"github.com/pion/webrtc/v3"
)

func TestWebRTCTunnel_DataTransfer(t *testing.T) {
	const containerID = "test-container-abc123"
	const portKey = "8080/tcp"

	// ── 1. 가짜 컨테이너 TCP 에코 서버 ──────────────────────────────────────
	// 실제 Docker 컨테이너 대신 로컬 TCP 서버로 대체합니다.
	// Worker의 ConnectWebRTCHandler는 DataChannel이 열리면 이 주소로 TCP 연결합니다.
	containerLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("fake container listen: %v", err)
	}
	defer containerLn.Close()
	containerPort := containerLn.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			conn, err := containerLn.Accept()
			if err != nil {
				return
			}
			go echoConn(conn)
		}
	}()

	// ── 2. Worker ContainerStore에 컨테이너 정보 등록 ────────────────────────
	// Docker inspect 없이 Store에 직접 등록하여 DockerCli 의존성 제거.
	store := worker.NewContainerStore()
	store.Put(&worker.ContainerInfo{
		ContainerID:  containerID,
		Name:         "fake-container",
		PortBindings: map[string]int{portKey: containerPort},
	})

	// ── 3. InMemoryBroker: RabbitMQ 대체 ─────────────────────────────────────
	broker := testutil.NewInMemoryBroker()

	// ── 4. WorkerAgent 시작 (goroutine) ──────────────────────────────────────
	const workerID = "test-worker-001"
	workerQueue := "worker-" + workerID

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()

	noSTUN := &webrtc.Configuration{} // STUN 없이 host candidate만 사용 (외부 네트워크 불필요)
	traffic := metrics.NewTrafficStore()

	webrtcHandler := &worker.ConnectWebRTCHandler{
		Store:        store,
		Broker:       broker,
		WorkerID:     workerID,
		DockerCli:    nil, // Store에 이미 등록되어 있으므로 Docker 불필요
		Traffic:      traffic,
		WebRTCConfig: noSTUN,
	}

	// WorkerAgent: main에서는 NewAgent(mq, workerID, heartbeatQueue)
	// 하트비트 큐는 테스트에서 무시됨 (nobody subscribes → Publish silently succeeds)
	const heartbeatQueue = "test-heartbeat"
	agent := worker.NewAgent(broker, workerID, heartbeatQueue)
	agent.Register("CONNECT_WEBRTC", webrtcHandler)
	go func() {
		if err := agent.Run(workerCtx, workerQueue, traffic); err != nil {
			t.Logf("[worker] agent.Run error: %v", err)
		}
	}()

	// ── 5. FakeHub 서버 시작 ──────────────────────────────────────────────────
	// Hub의 두 엔드포인트를 모킹합니다:
	//   GET  /api/v1/containers/{id}         → 컨테이너 상태 반환
	//   POST /api/v1/containers/{id}/connect → WebRTC 시그널링 중계
	fakeHub := newFakeHub(t, containerID, portKey, containerPort, broker, workerQueue)
	defer fakeHub.Close()

	// ── 6. Client Tunnel 생성 ─────────────────────────────────────────────────
	hubClient := &client.HubClient{
		BaseURL:    fakeHub.URL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}

	tunnel := &client.Tunnel{
		Config:       &client.Config{ContainerID: containerID},
		HubClient:    hubClient,
		WebRTCConfig: noSTUN,
	}

	// ── 7. WebRTC PeerConnection 수립 ─────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pc, err := tunnel.Connect(ctx)
	if err != nil {
		t.Fatalf("WebRTC connect failed: %v", err)
	}
	defer pc.Close()

	t.Logf("WebRTC connected (state: %s)", pc.ConnectionState())

	// ── 8. DataChannel 열어 에코 검증 ─────────────────────────────────────────
	// Worker 쪽 DataChannel 핸들러가 portKey를 label로 식별하므로 동일하게 설정.
	echoDone := make(chan string, 1)
	echoErr := make(chan error, 1)

	dc, err := pc.CreateDataChannel(portKey, nil)
	if err != nil {
		t.Fatalf("CreateDataChannel failed: %v", err)
	}

	dc.OnOpen(func() {
		raw, err := dc.Detach()
		if err != nil {
			echoErr <- fmt.Errorf("Detach failed: %w", err)
			return
		}
		const msg = "hello peercaas"
		if _, err := raw.Write([]byte(msg)); err != nil {
			echoErr <- fmt.Errorf("write failed: %w", err)
			return
		}
		buf := make([]byte, len(msg))
		if _, err := raw.Read(buf); err != nil {
			echoErr <- fmt.Errorf("read failed: %w", err)
			return
		}
		echoDone <- string(buf)
	})

	select {
	case received := <-echoDone:
		if received != "hello peercaas" {
			t.Errorf("echo mismatch: got %q, want %q", received, "hello peercaas")
		}
		t.Logf("DataChannel echo OK: %q", received)
	case err := <-echoErr:
		t.Fatalf("DataChannel error: %v", err)
	case <-ctx.Done():
		t.Fatal("timeout: WebRTC DataChannel echo did not complete in time")
	}
}

// ─── FakeHub ─────────────────────────────────────────────────────────────────

// newFakeHub는 Hub의 컨테이너 조회 + WebRTC 시그널링 엔드포인트를 모킹하는
// httptest.Server를 생성합니다.
//
// POST /api/v1/containers/{id}/connect 동작:
//  1. Client로부터 SDP offer 수신
//  2. InMemoryBroker를 통해 Worker에 CONNECT_WEBRTC 커맨드 발행
//  3. Worker의 SDP answer를 reply 큐에서 수신
//  4. Client에 answer 반환
func newFakeHub(
	t *testing.T,
	containerID, portKey string,
	containerPort int,
	broker core.Broker,
	workerQueue string,
) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// GET /api/v1/containers/{id}
	mux.HandleFunc("/api/v1/containers/"+containerID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]any{
			"code": 200,
			"data": map[string]any{
				"containerId":  containerID,
				"status":       "RUNNING",
				"workerId":     "test-worker-001",
				"portBindings": map[string]int{portKey: containerPort},
			},
		})
	})

	// POST /api/v1/containers/{id}/connect
	mux.HandleFunc("/api/v1/containers/"+containerID+"/connect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Offer webrtc.SessionDescription `json:"offer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("FakeHub: failed to decode connect request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// reply 큐를 구독한 뒤 Worker에 커맨드 발행
		replyQueue := fmt.Sprintf("webrtc.reply.%d", time.Now().UnixNano())
		handlerCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		replyCh, err := broker.Subscribe(handlerCtx, replyQueue)
		if err != nil {
			t.Errorf("FakeHub: Subscribe failed: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		payload, _ := json.Marshal(map[string]any{
			"containerId": containerID,
			"offer":       req.Offer,
			"replyQueue":  replyQueue,
		})
		cmd := core.CommandMessage{
			CmdType:   "CONNECT_WEBRTC",
			TraceID:   fmt.Sprintf("trace-%d", time.Now().UnixNano()),
			Payload:   json.RawMessage(payload),
			Timestamp: time.Now().Unix(),
		}
		cmdBytes, _ := json.Marshal(cmd)

		if err := broker.Publish(handlerCtx, workerQueue, cmdBytes); err != nil {
			t.Errorf("FakeHub: failed to publish CONNECT_WEBRTC: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Worker의 answer 대기
		select {
		case evt, ok := <-replyCh:
			if !ok {
				t.Errorf("FakeHub: reply channel closed unexpectedly")
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			var replyMsg core.CommandMessage
			if err := json.Unmarshal(evt.Payload(), &replyMsg); err != nil {
				t.Errorf("FakeHub: failed to unmarshal reply: %v", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			var answerPayload struct {
				ContainerID string                    `json:"containerId"`
				Answer      webrtc.SessionDescription `json:"answer"`
			}
			if err := json.Unmarshal(replyMsg.Payload, &answerPayload); err != nil {
				t.Errorf("FakeHub: failed to unmarshal answer payload: %v", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			evt.Ack()
			writeJSON(w, map[string]any{
				"code": 200,
				"data": map[string]any{"answer": answerPayload.Answer},
			})
		case <-handlerCtx.Done():
			t.Errorf("FakeHub: timeout waiting for WebRTC answer from worker")
			http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
		}
	})

	return httptest.NewServer(mux)
}

// ─── 헬퍼 ────────────────────────────────────────────────────────────────────

func echoConn(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		if _, err := conn.Write(buf[:n]); err != nil {
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
