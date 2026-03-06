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
	// Parse command line flags
	resetIP := flag.Bool("reset", false, "Reset the registered IP for this worker on the Hub")
	flag.Parse()

	// Load configuration
	cfg := config.Load("worker")
	if cfg.Worker.WorkerID == "" {
		log.Fatal("CRITICAL: WORKER_ID environment variable is missing")
	}

	// Initialize and call Hub Init API
	hubClient := worker.NewHubClient(cfg.Worker.HubURL)

	if *resetIP {
		log.Printf("Resetting IP for worker [%s] on Hub...", cfg.Worker.WorkerID)
		if err := hubClient.ResetWorkerIP(cfg.Worker.WorkerID, cfg.Worker.WorkerKey); err != nil {
			log.Fatalf("CRITICAL: Failed to reset IP on hub: %v", err)
		}
		log.Println("Successfully reset IP. Exiting.")
		return
	}

	if err := hubClient.InitializeWorker(cfg.Worker.WorkerID, cfg.Worker.WorkerKey); err != nil {
		log.Fatalf("CRITICAL: Failed to initialize worker with hub: %v", err)
	}

	// Initialize infrastructure
	broker := rabbitmq.New(cfg.RabbitMQ.GetURL())
	if err := broker.Connect(); err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer broker.Close()

	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerCli.Close()

	// Initialize Metrics Persistence & Shipping
	repo, err := metrics.NewMetricRepository("metrics.db")
	if err != nil {
		log.Fatalf("Failed to initialize SQLite: %v", err)
	}
	defer repo.Close()

	// Use default local VM endpoint if not configured
	vmURL := cfg.Worker.VMURL
	if vmURL == "" {
		vmURL = "http://localhost:8428/write"
	}
	shipper := metrics.NewMetricShipper(vmURL, cfg.Worker.VMUser, cfg.Worker.VMPass)

	// Initialize shared services
	store := worker.NewContainerStore()
	traffic := metrics.NewTrafficStore()
	latency := metrics.NewLatencyMeasurer()
	publisher := &worker.BrokerResultPublisher{
		Broker:    broker,
		QueueName: cfg.Worker.ResultQueue,
	}

	// Initialize Agent
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

	// Register command handlers
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

	// Graceful shutdown context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("=== PeerCaaS Worker Agent [%s] Ready ===", cfg.Worker.WorkerID)

	// Run the agent
	if err := agent.Run(ctx, cfg.Worker.WorkerID, traffic, latency); err != nil {
		log.Printf("Worker terminated with error: %v", err)
	}
}
