package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"agents/internal/app/client"
	"agents/internal/metrics"
)

func main() {
	// 1. Load configuration
	cfg, err := client.LoadConfig()
	if err != nil {
		log.Fatalf("CRITICAL: Configuration error: %v", err)
	}

	// 2. Initialize components
	hubClient := client.NewHubClient(cfg.HubURL)
	traffic := metrics.NewTrafficStore()

	// 3. Setup context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Start background reporter
	metrics.StartReporter(ctx, cfg.HubURL, "client", cfg.ClientKey, traffic)

	log.Println("=== PeerCaaS Client Agent ===")
	log.Printf("[Main] Container: %s | Hub: %s", cfg.ContainerID, cfg.HubURL)

	// 5. Run connection manager
	manager := client.NewConnectionManager(cfg, hubClient, traffic)
	if err := manager.Run(ctx); err != nil {
		log.Fatalf("CRITICAL: ConnectionManager stopped unexpectedly: %v", err)
	}
}
