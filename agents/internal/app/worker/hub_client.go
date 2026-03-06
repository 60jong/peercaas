package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HubClient provides a client for communicating with the central PeerCaaS Hub.
type HubClient struct {
	hubURL string
	client *http.Client
}

// NewHubClient initializes a new HubClient with a standard timeout.
func NewHubClient(hubURL string) *HubClient {
	return &HubClient{
		hubURL: hubURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// WorkerInitRequest defines the registration payload for a worker node.
type WorkerInitRequest struct {
	WorkerID  string `json:"workerId"`
	WorkerKey string `json:"workerKey"`
}

// BaseResponse represents the standard API response structure from the Hub.
type BaseResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// InitializeWorker registers the worker with the Hub. 
// It ensures the worker's ID and Key are valid and maps the worker's current IP address.
func (c *HubClient) InitializeWorker(workerID, workerKey string) error {
	url := fmt.Sprintf("%s/api/v1/agent/worker/init", c.hubURL)
	reqBody, err := json.Marshal(WorkerInitRequest{
		WorkerID:  workerID,
		WorkerKey: workerKey,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal init request: %w", err)
	}

	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to call hub init API: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		var errResp BaseResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("Status: %s, Reason: %s", resp.Status, errResp.Message)
		}
		return fmt.Errorf("Status: %s", resp.Status)
	}

	return nil
}

// ResetWorkerIP clears the registered IP mapping for this worker on the Hub.
// This is required if the worker's public IP changes.
func (c *HubClient) ResetWorkerIP(workerID, workerKey string) error {
	url := fmt.Sprintf("%s/api/v1/agent/worker/ip?workerId=%s&workerKey=%s", c.hubURL, workerID, workerKey)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call hub reset IP API: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		var errResp BaseResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("Status: %s, Reason: %s", resp.Status, errResp.Message)
		}
		return fmt.Errorf("Status: %s", resp.Status)
	}

	return nil
}
