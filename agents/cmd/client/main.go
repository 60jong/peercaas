package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agents/internal/app/client"
)

func main() {
	cfg := client.LoadConfig()

	hubClient := &client.HubClient{
		BaseURL:    cfg.HubURL,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("=== PeerCaaS Client Agent ===")
	log.Printf("Container: %s | Hub: %s", cfg.ContainerID, cfg.HubURL)

	manager := client.NewConnectionManager(cfg, hubClient)
	if err := manager.Run(ctx); err != nil {
		log.Fatalf("ConnectionManager failed: %v", err)
	}
}
