package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pion/webrtc/v3"
)

// HubClient provides methods to interact with the central Hub for container and connection metadata.
type HubClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewHubClient initializes a HubClient with a standard timeout.
func NewHubClient(baseURL string) *HubClient {
	return &HubClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *HubClient) decodeResponse(resp *http.Response, target any) error {
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode hub response: %w", err)
	}

	if apiResp.Code >= 400 {
		return fmt.Errorf("hub error %d [%s]: %s", apiResp.Code, apiResp.Message, string(apiResp.Data))
	}

	if target != nil {
		if err := json.Unmarshal(apiResp.Data, target); err != nil {
			return fmt.Errorf("failed to unmarshal data field: %w", err)
		}
	}

	return nil
}

// ContainerInfoResponse describes the metadata returned by the Hub for a specific container.
type ContainerInfoResponse struct {
	ContainerID  string         `json:"containerId"`
	Status       string         `json:"status"`
	WorkerID     string         `json:"workerId"`
	PortBindings map[string]int `json:"portBindings"`
}

// GetContainerInfo retrieves metadata for a specific container from the Hub.
func (c *HubClient) GetContainerInfo(ctx context.Context, containerID string) (*ContainerInfoResponse, error) {
	url := fmt.Sprintf("%s/api/v1/containers/%s", c.BaseURL, containerID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	var info ContainerInfoResponse
	if err := c.decodeResponse(resp, &info); err != nil {
		return nil, err
	}

	if info.Status != "RUNNING" {
		return nil, fmt.Errorf("container %s is not running (status: %s)", containerID, info.Status)
	}

	return &info, nil
}

// SignalConnect performs the WebRTC SDP exchange via the Hub signaling channel.
func (c *HubClient) SignalConnect(ctx context.Context, containerID string, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	url := fmt.Sprintf("%s/api/v1/containers/%s/connect", c.BaseURL, containerID)

	body, _ := json.Marshal(map[string]any{"offer": offer})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Answer webrtc.SessionDescription `json:"answer"`
	}
	if err := c.decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result.Answer, nil
}

// RelayResponse describes the session details for a relay connection.
type RelayResponse struct {
	RelayHost string `json:"relayHost"`
	RelayPort int    `json:"relayPort"`
	Token     string `json:"token"`
	PortKey   string `json:"portKey"`
}

// RequestRelay requests a fallback TCP relay session from the Hub.
func (c *HubClient) RequestRelay(ctx context.Context, containerID string, portKey string) (*RelayResponse, error) {
	url := fmt.Sprintf("%s/api/v1/containers/%s/relay", c.BaseURL, containerID)

	body, _ := json.Marshal(map[string]string{"portKey": portKey})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	var relay RelayResponse
	if err := c.decodeResponse(resp, &relay); err != nil {
		return nil, err
	}

	return &relay, nil
}
