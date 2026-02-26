package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"agents/internal/app/worker"
	"agents/internal/config"
	"agents/internal/infra/mq/rabbitmq"
	"agents/internal/metrics"

	"github.com/docker/docker/client"
)

func main() {
	// 1. Worker Agent 설정 로드
	cfg := config.Load("worker")

	if cfg.Worker.WorkerID == "" {
		log.Fatal("CRITICAL ERROR: WORKER_ID environment variable is missing!")
	}

	queueName := cfg.Worker.WorkerID
	log.Printf("=== Starting Worker Agent ===")
	log.Printf("ID: %s", cfg.Worker.WorkerID)
	log.Printf("Target Queue: %s", queueName)

	// 2. RabbitMQ 연결
	broker := rabbitmq.New(cfg.RabbitMQ.GetURL())
	if err := broker.Connect(); err != nil {
		log.Fatal(err)
	}
	defer broker.Close()

	// 3. Docker 클라이언트 생성
	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerCli.Close()

	log.Printf("Starting Worker Agent: %s", cfg.Server.Name)
	agent := worker.NewAgent(broker, cfg.Worker.WorkerID, "peercaas.worker.heartbeat")

	// 4. Publisher 생성
	publisher := &worker.BrokerResultPublisher{
		Broker:    broker,
		QueueName: cfg.Worker.ResultQueue,
	}

	// 5. ContainerStore + TrafficStore + LatencyMeasurer 생성
	store := worker.NewContainerStore()
	traffic := metrics.NewTrafficStore()
	latency := metrics.NewLatencyMeasurer(cfg.Worker.HubURL)

	// 6. 핸들러 등록
	agent.Register("CREATE_CONTAINER", &worker.CreateContainerHandler{
		DockerCli: dockerCli,
		Publisher: publisher,
		WorkerId:  cfg.Worker.WorkerID,
		Store:     store,
	})
	agent.Register("DELETE_CONTAINER", &worker.DeleteContainerHandler{
		DockerCli: dockerCli,
		Store:     store,
	})
	agent.Register("CONNECT_WEBRTC", &worker.ConnectWebRTCHandler{
		Store:     store,
		Broker:    broker,
		WorkerID:  cfg.Worker.WorkerID,
		DockerCli: dockerCli,
		Traffic:   traffic,
	})
	agent.Register("RELAY_CONNECT", &worker.RelayConnectHandler{
		Store:     store,
		DockerCli: dockerCli,
		Traffic:   traffic,
	})

	// 7. 실행 (Graceful Shutdown 준비)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 8. Latency 측정 시작
	go latency.Start(ctx)

	log.Printf("=== PeerCaaS Worker Agent === (Unified Metrics via RMQ)")

	if err := agent.Run(ctx, queueName, traffic, latency); err != nil {
		log.Printf("Worker terminated with error: %v", err)
	}
}
