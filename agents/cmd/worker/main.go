package main

import (
	"context"
	"flag"
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
	// 1. Parse command line flags
	resetIP := flag.Bool("reset", false, "Reset the registered IP for this worker on the Hub")
	flag.Parse()

	// 2. Load configuration
	cfg, err := config.Load("worker")
	if err != nil {
		log.Fatalf("CRITICAL: Failed to load configuration: %v", err)
	}

	// 3. Generate or validate WorkerID
	if cfg.Worker.WorkerID == "" {
		if cfg.Worker.WorkerKey == "" {
			log.Fatal("CRITICAL: Both WORKER_ID and WORKER_KEY are missing. Provide at least WORKER_KEY.")
		}
		cfg.Worker.WorkerID = cfg.Worker.GenerateWorkerID()
		log.Printf("[Main] Automatically generated WorkerID from Key: %s", cfg.Worker.WorkerID)
	}

	// 4. Initialize Hub connection
	hubClient := worker.NewHubClient(cfg.Worker.HubURL)

	if *resetIP {
		log.Printf("[Main] Resetting IP for worker %s on Hub...", cfg.Worker.WorkerID)
		if err := hubClient.ResetWorkerIP(cfg.Worker.WorkerID, cfg.Worker.WorkerKey); err != nil {
			log.Fatalf("CRITICAL: Failed to reset IP: %v", err)
		}
		log.Println("[Main] Successfully reset IP. Exiting.")
		return
	}

	if err := hubClient.InitializeWorker(cfg.Worker.WorkerID, cfg.Worker.WorkerKey); err != nil {
		log.Fatalf("CRITICAL: Worker initialization failed: %v", err)
	}

	// 4. Initialize Infrastructure
	broker := rabbitmq.NewBroker(cfg.RabbitMQ.GetURL())
	if err := broker.Connect(); err != nil {
		log.Fatalf("CRITICAL: Failed to connect to RabbitMQ: %v", err)
	}
	defer broker.Close()

	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("CRITICAL: Failed to create Docker client: %v", err)
	}
	defer dockerCli.Close()

	repo, err := metrics.NewMetricRepository("metrics.db")
	if err != nil {
		log.Fatalf("CRITICAL: Failed to initialize metrics database: %v", err)
	}
	defer repo.Close()

	vmURL := cfg.Worker.VMURL
	if vmURL == "" {
		vmURL = "http://localhost:8428/write"
	}
	shipper := metrics.NewMetricShipper(vmURL, cfg.Worker.VMUser, cfg.Worker.VMPass)

	// 5. Initialize Services
	store := worker.NewContainerStore()
	traffic := metrics.NewTrafficStore()
	latency := metrics.NewLatencyMeasurer()
	publisher := &worker.BrokerResultPublisher{
		Broker:    broker,
		QueueName: cfg.Worker.ResultQueue,
	}

	// 6. Initialize Agent
	agent := worker.NewAgent(
		broker,
		cfg.Worker,
		"peercaas.worker.heartbeat",
		dockerCli,
		traffic,
		latency,
		repo,
		shipper,
		store,
	)

	// 7. Register Command Handlers
	registerHandlers(agent, dockerCli, publisher, store, traffic, broker, cfg.Worker.WorkerID)

	// 8. Run Agent with graceful shutdown support
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("=== PeerCaaS Worker Agent [%s] Ready ===", cfg.Worker.WorkerID)

	if err := agent.Run(ctx, cfg.Worker.WorkerID, traffic, latency); err != nil {
		log.Printf("[Main] Worker stopped with error: %v", err)
	}
}

func registerHandlers(agent *worker.WorkerAgent, dockerCli *client.Client, publisher worker.ResultPublisher, store *worker.ContainerStore, traffic *metrics.TrafficStore, broker *rabbitmq.RabbitMQBroker, workerID string) {
	agent.Register("CREATE_CONTAINER", &worker.CreateContainerHandler{
		DockerCli: dockerCli,
		Publisher: publisher,
		WorkerID:  workerID,
		Store:     store,
	})
	agent.Register("DELETE_CONTAINER", &worker.DeleteContainerHandler{
		DockerCli: dockerCli,
		Store:     store,
	})
	agent.Register("CONNECT_WEBRTC", &worker.ConnectWebRTCHandler{
		Store:     store,
		Broker:    broker,
		WorkerID:  workerID,
		DockerCli: dockerCli,
		Traffic:   traffic,
	})
	agent.Register("RELAY_CONNECT", &worker.RelayConnectHandler{
		Store:     store,
		DockerCli: dockerCli,
		Traffic:   traffic,
	})
}
