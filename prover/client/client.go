package client

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"

	"scroll-tech/prover/config"
)

// CoordinatorClient is a client used for interacting with the Coordinator service.
type CoordinatorClient struct {
	client *resty.Client
}

// NewCoordinatorClient constructs a new CoordinatorClient.
func NewCoordinatorClient(cfg *config.CoordinatorConfig) (*CoordinatorClient, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base URL is not specified")
	}

	client := resty.New().
		SetTimeout(time.Duration(cfg.Timeout) * time.Second).
		SetRetryCount(cfg.RetryCount).
		SetRetryWaitTime(time.Duration(cfg.RetryWaitTime) * time.Second).
		SetBaseURL(cfg.BaseURL)

	return &CoordinatorClient{
		client: client,
	}, nil
}

// Login sends login request to the coordinator.
func (c *CoordinatorClient) Login(ctx context.Context, req *ProverLoginRequest) (*ProverLoginResponse, error) {
	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&ProverLoginResponse{}).
		Post("/api/login")

	if err != nil {
		return nil, fmt.Errorf("login failed: %v", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to login, status code: %v", resp.StatusCode())
	}

	result := resp.Result().(*ProverLoginResponse)

	if result.ErrCode != 200 {
		return nil, fmt.Errorf("failed to login, error code: %v, error message: %v", result.ErrCode, result.ErrMsg)
	}

	// store JWT token for future requests
	c.client.SetAuthToken(result.Data.Token)

	return result, nil
}

// ProverTasks sends a request to the coordinator to get prover tasks.
func (c *CoordinatorClient) ProverTasks(ctx context.Context, req *ProverTasksRequest) (*ProverTasksResponse, error) {
	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&ProverTasksResponse{}).
		Post("/api/prover_tasks")

	if err != nil {
		return nil, fmt.Errorf("request for ProverTasks failed: %v", err)
	}

	result := resp.Result().(*ProverTasksResponse)

	if result.ErrCode != 200 {
		return nil, fmt.Errorf("error code: %v, error message: %v", result.ErrCode, result.ErrMsg)
	}

	return result, nil
}

// SubmitProof sends a request to the coordinator to submit proof.
func (c *CoordinatorClient) SubmitProof(ctx context.Context, req *SubmitProofRequest) (*SubmitProofResponse, error) {
	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&SubmitProofResponse{}).
		Post("/coordinator/v1/submit_proof")

	if err != nil {
		return nil, fmt.Errorf("submit proof request failed: %v", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to submit proof, status code: %v", resp.StatusCode())
	}

	result := resp.Result().(*SubmitProofResponse)

	if result.ErrCode != 200 {
		return nil, fmt.Errorf("error code: %v, error message: %v", result.ErrCode, result.ErrMsg)
	}

	return result, nil
}
