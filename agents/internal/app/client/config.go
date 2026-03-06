package client

import (
	"fmt"
	"os"
)

const defaultHubURL = "http://localhost:8080"

// Config holds the configuration for the client agent.
type Config struct {
	ContainerID string
	ClientKey   string
	HubURL      string
}

// LoadConfig reads configuration from environment variables.
// It returns an error if required variables are missing.
func LoadConfig() (*Config, error) {
	containerID := os.Getenv("CONTAINER_ID")
	if containerID == "" {
		return nil, fmt.Errorf("CONTAINER_ID environment variable is required")
	}

	clientKey := os.Getenv("PEERCAAS_CLIENT_KEY")
	if clientKey == "" {
		clientKey = containerID
	}

	hubURL := os.Getenv("HUB_URL")
	if hubURL == "" {
		hubURL = defaultHubURL
	}

	return &Config{
		ContainerID: containerID,
		ClientKey:   clientKey,
		HubURL:      hubURL,
	}, nil
}
