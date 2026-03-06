package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/streadway/amqp"
)

// ContainerRequest represents the creation parameters received from RabbitMQ.
type ContainerRequest struct {
	Repository string   `json:"repository"`
	Image      string   `json:"image"`
	Tag        string   `json:"tag"`
	Ports      []string `json:"ports"`
	Envs       []string `json:"envs"`
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer cli.Close()

	conn, err := amqp.Dial("amqp://root:991911@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	msgs, err := ch.Consume("worker_tasks", "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	fmt.Println("🚀 Worker Agent Test Script: Waiting for messages...")

	for d := range msgs {
		var req ContainerRequest
		if err := json.Unmarshal(d.Body, &req); err != nil {
			log.Printf("❌ Message parsing error: %v", err)
			continue
		}

		fmt.Printf("📦 Received creation request: %s:%s\n", req.Image, req.Tag)
		if err := createAndRunContainer(cli, req); err != nil {
			log.Printf("❌ Container execution failed: %v", err)
		}
	}
}

func createAndRunContainer(cli *client.Client, req ContainerRequest) error {
	ctx := context.Background()
	fullImageName := fmt.Sprintf("%s:%s", req.Image, req.Tag)

	log.Printf("Pulling image: %s", fullImageName)
	reader, err := cli.ImagePull(ctx, fullImageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull error: %w", err)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)

	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}

	for _, portStr := range req.Ports {
		hostPort, containerPort := splitPort(portStr)
		if hostPort == "" || containerPort == "" {
			continue
		}
		cPort := nat.Port(containerPort + "/tcp")
		portBindings[cPort] = []nat.PortBinding{{HostPort: hostPort}}
		exposedPorts[cPort] = struct{}{}
	}

	config := &container.Config{
		Image:        fullImageName,
		Env:          req.Envs,
		ExposedPorts: exposedPorts,
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("create error: %w", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start error: %w", err)
	}

	fmt.Printf("✅ Container started successfully! ID: %s\n", resp.ID[:12])
	return nil
}

func splitPort(portStr string) (string, string) {
	parts := strings.Split(portStr, ":")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
