package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings" // 문자열 처리를 위해 추가

	"github.com/docker/docker/api/types/container" // [수정] 컨테이너 옵션 패키지
	"github.com/docker/docker/api/types/image"     // [수정] 이미지 옵션 패키지
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/streadway/amqp"
)

// RMQ로부터 받을 메시지 구조체
type ContainerRequest struct {
	Repository string   `json:"repository"`
	Image      string   `json:"image"`
	Tag        string   `json:"tag"`
	Ports      []string `json:"ports"`
	Envs       []string `json:"envs"`
}

func main() {
	// ... 기존 main 함수 로직 동일 ...
	// (RabbitMQ 연결 부분은 그대로 두셔도 됩니다)

	// 테스트용으로 바로 호출해보고 싶다면 아래 주석을 풀고 테스트하세요
	// cli, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	// req := ContainerRequest{Image: "mysql", Tag: "8", Ports: []string{"3306:3306"}, Envs: []string{"MYSQL_ROOT_PASSWORD=1234"}}
	// createAndRunContainer(cli, req)

	// 원래 main 로직 유지
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	// 주의: DinD 환경에서 실행 시 localhost는 DinD 컨테이너 자체를 의미합니다.
	// 호스트의 RabbitMQ에 붙으려면 외부 IP(58.143...)를 사용해야 할 수 있습니다.
	conn, err := amqp.Dial("amqp://root:991911@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	msgs, err := ch.Consume("worker_tasks", "", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("🚀 Worker Agent 시작: 메시지 대기 중...")

	for d := range msgs {
		var req ContainerRequest
		if err := json.Unmarshal(d.Body, &req); err != nil {
			log.Printf("❌ 메시지 파싱 에러: %v", err)
			continue
		}

		fmt.Printf("📦 컨테이너 생성 요청 수신: %s:%s\n", req.Image, req.Tag)
		err := createAndRunContainer(cli, req)
		if err != nil {
			log.Printf("❌ 컨테이너 실행 실패: %v", err)
		}
	}
}

func createAndRunContainer(cli *client.Client, req ContainerRequest) error {
	ctx := context.Background()
	fullImageName := fmt.Sprintf("%s:%s", req.Image, req.Tag)

	// [수정 1] types.ImagePullOptions -> image.PullOptions
	reader, err := cli.ImagePull(ctx, fullImageName, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	// io.Copy(os.Stdout, reader) // 로그가 너무 길면 주석 처리

	// 포트 설정
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

	// 컨테이너 설정
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
		return err
	}

	// [수정 2] types.ContainerStartOptions -> container.StartOptions
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}

	fmt.Printf("✅ 컨테이너 실행 완료! ID: %s\n", resp.ID)
	return nil
}

// [보완] Sscanf는 포맷이 조금만 달라도 실패하므로 strings.Split이 더 안전합니다.
func splitPort(portStr string) (string, string) {
	parts := strings.Split(portStr, ":")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
