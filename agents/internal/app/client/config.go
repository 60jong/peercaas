package client

import (
	"log"
	"os"
)

const defaultHubURL = "http://localhost:8080"

type Config struct {
	ContainerID string // env: CONTAINER_ID (required)
	HubURL      string // env: HUB_URL (default: http://localhost:8080)
}

func LoadConfig() *Config {
	containerID := os.Getenv("CONTAINER_ID")
	if containerID == "" {
		log.Fatal("CONTAINER_ID environment variable is required")
	}

	hubURL := os.Getenv("HUB_URL")
	if hubURL == "" {
		hubURL = defaultHubURL
	}

	return &Config{
		ContainerID: containerID,
		HubURL:      hubURL,
	}
}
