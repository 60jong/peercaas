package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pion/webrtc/v3"
)

type HubClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// Hub는 모든 응답을 ApiResponse로 감싸서 반환
// 성공: { "code": 200, "data": { ... } }
// 실패: { "code": 4xx/5xx, "message": "ERROR_CODE", "data": "에러메시지" }
type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func parseResponse(body []byte) (apiResponse, error) {
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, fmt.Errorf("failed to parse hub response: %w", err)
	}
	if resp.Code >= 400 {
		return resp, fmt.Errorf("hub error %d [%s]: %s", resp.Code, resp.Message, string(resp.Data))
	}
	return resp, nil
}

type ContainerInfoResponse struct {
	ContainerID  string         `json:"containerId"`
	Status       string         `json:"status"`
	WorkerID     string         `json:"workerId"`
	PortBindings map[string]int `json:"portBindings"`
}

type ConnectRequest struct {
	Offer webrtc.SessionDescription `json:"offer"`
}

type ConnectResponse struct {
	Answer webrtc.SessionDescription `json:"answer"`
}

type RelayRequest struct {
	PortKey string `json:"portKey"`
}

type RelayResponse struct {
	RelayHost string `json:"relayHost"`
	RelayPort int    `json:"relayPort"`
	Token     string `json:"token"`
	PortKey   string `json:"portKey"`
}

func (c *HubClient) GetContainerInfo(ctx context.Context, containerID string) (*ContainerInfoResponse, error) {
	url := fmt.Sprintf("%s/api/v1/containers/%s", c.BaseURL, containerID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	apiResp, err := parseResponse(body)
	if err != nil {
		return nil, err
	}

	var info ContainerInfoResponse
	if err := json.Unmarshal(apiResp.Data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container info: %w", err)
	}

	if info.Status != "RUNNING" {
		return nil, fmt.Errorf("container %s is not running (status: %s)", containerID, info.Status)
	}

	return &info, nil
}

func (c *HubClient) SignalConnect(ctx context.Context, containerID string, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	url := fmt.Sprintf("%s/api/v1/containers/%s/connect", c.BaseURL, containerID)

	reqBody, err := json.Marshal(ConnectRequest{Offer: offer})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connect request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to signal connect: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	apiResp, err := parseResponse(body)
	if err != nil {
		return nil, err
	}

	var connectResp ConnectResponse
	if err := json.Unmarshal(apiResp.Data, &connectResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal connect response: %w", err)
	}

	return &connectResp.Answer, nil
}

// RequestRelay: WebRTC 실패 시 Engine TCP relay 세션을 Hub에 요청
func (c *HubClient) RequestRelay(ctx context.Context, containerID string, portKey string) (*RelayResponse, error) {
	url := fmt.Sprintf("%s/api/v1/containers/%s/relay", c.BaseURL, containerID)

	reqBody, err := json.Marshal(RelayRequest{PortKey: portKey})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal relay request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request relay: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	apiResp, err := parseResponse(body)
	if err != nil {
		return nil, err
	}

	var relayResp RelayResponse
	if err := json.Unmarshal(apiResp.Data, &relayResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relay response: %w", err)
	}

	return &relayResp, nil
}
