package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HubClient struct {
	hubURL string
	client *http.Client
}

func NewHubClient(hubURL string) *HubClient {
	return &HubClient{
		hubURL: hubURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type WorkerInitRequest struct {
	WorkerID  string `json:"workerId"`
	WorkerKey string `json:"workerKey"`
}

type BaseResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *HubClient) InitializeWorker(workerID, workerKey string) error {
	url := fmt.Sprintf("%s/api/v1/agent/worker/init", c.hubURL)
	reqBody, _ := json.Marshal(WorkerInitRequest{
		WorkerID:  workerID,
		WorkerKey: workerKey,
	})

	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to call hub init API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp BaseResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("Status: %s, Reason: %s", resp.Status, errResp.Message)
		}
		return fmt.Errorf("Status: %s", resp.Status)
	}

	return nil
}

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp BaseResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("Status: %s, Reason: %s", resp.Status, errResp.Message)
		}
		return fmt.Errorf("Status: %s", resp.Status)
	}

	return nil
}

