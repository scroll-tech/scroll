package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"

	"scroll-tech/prover/config"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

// CoordinatorClient is a client used for interacting with the Coordinator service.
type CoordinatorClient struct {
	client *resty.Client

	proverName string
	priv       *ecdsa.PrivateKey
}

// NewCoordinatorClient constructs a new CoordinatorClient.
func NewCoordinatorClient(cfg *config.CoordinatorConfig, proverName string, priv *ecdsa.PrivateKey) (*CoordinatorClient, error) {
	client := resty.New().
		SetTimeout(time.Duration(cfg.ConnectionTimeoutSec) * time.Second).
		SetRetryCount(cfg.RetryCount).
		SetRetryWaitTime(time.Duration(cfg.RetryWaitTimeSec) * time.Second).
		SetBaseURL(cfg.BaseURL)

	return &CoordinatorClient{
		client:     client,
		proverName: proverName,
		priv:       priv,
	}, nil
}

// Login completes the entire login process in one function call.
func (c *CoordinatorClient) Login(ctx context.Context) error {
	// Get random string
	randomResp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetResult(&RandomResponse{}).
		Get("/v1/random")

	if err != nil {
		return fmt.Errorf("get random string failed: %v", err)
	}

	if randomResp.StatusCode() != 200 {
		return fmt.Errorf("failed to get random string, status code: %v", randomResp.StatusCode())
	}

	randomResult := randomResp.Result().(*RandomResponse)

	// Prepare and sign the login request
	identity := &message.Identity{
		ProverName: c.proverName,
		Random:     randomResult.Random,
	}

	authMsg := &message.AuthMsg{
		Identity: identity,
	}

	err = authMsg.SignWithKey(c.priv)
	if err != nil {
		return fmt.Errorf("signature failed: %v", err)
	}

	// Login to coordinator
	loginReq := &LoginRequest{
		Message: *authMsg,
	}

	loginResp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(loginReq).
		SetResult(&LoginResponse{}).
		Post("/v1/login")

	if err != nil {
		return fmt.Errorf("login failed: %v", err)
	}

	if loginResp.StatusCode() != 200 {
		return fmt.Errorf("failed to login, status code: %v", loginResp.StatusCode())
	}

	loginResult := loginResp.Result().(*LoginResponse)

	if loginResult.ErrCode != types.Success {
		return fmt.Errorf("failed to login, error code: %v, error message: %v", loginResult.ErrCode, loginResult.ErrMsg)
	}

	// Store JWT token for future requests
	c.client.SetAuthToken(loginResult.Data.Token)

	return nil
}

// GetTask sends a request to the coordinator to get prover task.
func (c *CoordinatorClient) GetTask(ctx context.Context, req *GetTaskRequest) (*GetTaskResponse, error) {
	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&GetTaskResponse{}).
		Post("/v1/get_task")

	if err != nil {
		return nil, fmt.Errorf("request for GetTask failed: %v", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get task, status code: %v", resp.StatusCode())
	}

	result := resp.Result().(*GetTaskResponse)

	if result.ErrCode != types.Success {
		if result.ErrCode == types.ErrJWTTokenExpired {
			if err := c.Login(ctx); err != nil {
				return nil, fmt.Errorf("JWT expired, re-login failed: %v", err)
			}
			return c.GetTask(ctx, req)
		}
		return nil, fmt.Errorf("error code: %v, error message: %v", result.ErrCode, result.ErrMsg)
	}

	return result, nil
}

// SubmitProof sends a request to the coordinator to submit proof.
func (c *CoordinatorClient) SubmitProof(ctx context.Context, req *SubmitProofRequest) error {
	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&SubmitProofResponse{}).
		Post("/v1/submit_proof")

	if err != nil {
		return fmt.Errorf("submit proof request failed: %v", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("failed to submit proof, status code: %v", resp.StatusCode())
	}

	result := resp.Result().(*SubmitProofResponse)

	if result.ErrCode != types.Success {
		if result.ErrCode == types.ErrJWTTokenExpired {
			if err := c.Login(ctx); err != nil {
				return fmt.Errorf("JWT expired, re-login failed: %v", err)
			}
			return c.SubmitProof(ctx, req)
		}
		return fmt.Errorf("error code: %v, error message: %v", result.ErrCode, result.ErrMsg)
	}
	return nil
}
