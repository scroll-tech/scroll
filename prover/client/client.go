package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/prover/config"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/version"
)

// CoordinatorClient is a client used for interacting with the Coordinator service.
type CoordinatorClient struct {
	client *resty.Client

	proverName   string
	hardForkName string
	priv         *ecdsa.PrivateKey

	mu sync.Mutex
}

// NewCoordinatorClient constructs a new CoordinatorClient.
func NewCoordinatorClient(cfg *config.CoordinatorConfig, proverName string, hardForkName string, priv *ecdsa.PrivateKey) (*CoordinatorClient, error) {
	client := resty.New().
		SetTimeout(time.Duration(cfg.ConnectionTimeoutSec) * time.Second).
		SetRetryCount(cfg.RetryCount).
		SetRetryWaitTime(time.Duration(cfg.RetryWaitTimeSec) * time.Second).
		SetBaseURL(cfg.BaseURL).
		AddRetryAfterErrorCondition().
		AddRetryCondition(func(response *resty.Response, err error) bool {
			if err != nil {
				log.Warn("Encountered an error while sending the request. Retrying...", "error", err)
				return true
			}
			return response.IsError()
		})

	log.Info("successfully initialized prover client",
		"base url", cfg.BaseURL,
		"connection timeout (second)", cfg.ConnectionTimeoutSec,
		"retry count", cfg.RetryCount,
		"retry wait time (second)", cfg.RetryWaitTimeSec)

	return &CoordinatorClient{
		client:       client,
		proverName:   proverName,
		hardForkName: hardForkName,
		priv:         priv,
	}, nil
}

// Login completes the entire login process in one function call.
func (c *CoordinatorClient) Login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var challengeResult ChallengeResponse

	// Get random string
	challengeResp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetResult(&challengeResult).
		Get("/coordinator/v1/challenge")

	if err != nil {
		return fmt.Errorf("get random string failed: %w", err)
	}

	if challengeResp.StatusCode() != 200 {
		return fmt.Errorf("failed to get random string, status code: %v", challengeResp.StatusCode())
	}

	// Prepare and sign the login request
	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			ProverVersion: version.Version,
			ProverName:    c.proverName,
			Challenge:     challengeResult.Data.Token,
			HardForkName:  c.hardForkName,
		},
	}

	err = authMsg.SignWithKey(c.priv)
	if err != nil {
		return fmt.Errorf("signature failed: %w", err)
	}

	// Login to coordinator
	loginReq := &LoginRequest{
		Message: struct {
			Challenge     string `json:"challenge"`
			ProverName    string `json:"prover_name"`
			ProverVersion string `json:"prover_version"`
			HardForkName  string `json:"hard_fork_name"`
		}{
			Challenge:     authMsg.Identity.Challenge,
			ProverName:    authMsg.Identity.ProverName,
			ProverVersion: authMsg.Identity.ProverVersion,
			HardForkName:  authMsg.Identity.HardForkName,
		},
		Signature: authMsg.Signature,
	}

	// store JWT token for login requests
	c.client.SetAuthToken(challengeResult.Data.Token)

	var loginResult LoginResponse
	loginResp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(loginReq).
		SetResult(&loginResult).
		Post("/coordinator/v1/login")

	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if loginResp.StatusCode() != 200 {
		return fmt.Errorf("failed to login, status code: %v", loginResp.StatusCode())
	}

	if loginResult.ErrCode != types.Success {
		return fmt.Errorf("failed to login, error code: %v, error message: %v", loginResult.ErrCode, loginResult.ErrMsg)
	}

	// store JWT token for future requests
	c.client.SetAuthToken(loginResult.Data.Token)

	return nil
}

// GetTask sends a request to the coordinator to get prover task.
func (c *CoordinatorClient) GetTask(ctx context.Context, req *GetTaskRequest) (*GetTaskResponse, error) {
	var result GetTaskResponse

	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&result).
		Post("/coordinator/v1/get_task")

	if err != nil {
		return nil, fmt.Errorf("request for GetTask failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get task, status code: %v", resp.StatusCode())
	}

	if result.ErrCode == types.ErrJWTTokenExpired {
		log.Info("JWT expired, attempting to re-login")
		if err := c.Login(ctx); err != nil {
			return nil, fmt.Errorf("JWT expired, re-login failed: %w", err)
		}
		log.Info("re-login success")
		return c.GetTask(ctx, req)
	}
	if result.ErrCode != types.Success {
		return nil, fmt.Errorf("error code: %v, error message: %v", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}

// SubmitProof sends a request to the coordinator to submit proof.
func (c *CoordinatorClient) SubmitProof(ctx context.Context, req *SubmitProofRequest) error {
	var result SubmitProofResponse

	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&result).
		Post("/coordinator/v1/submit_proof")

	if err != nil {
		log.Error("submit proof request failed", "error", err)
		return fmt.Errorf("submit proof request failed: %w", ErrCoordinatorConnect)
	}

	if resp.StatusCode() != 200 {
		log.Error("failed to submit proof", "status code", resp.StatusCode())
		return fmt.Errorf("failed to submit proof, status code not 200: %w", ErrCoordinatorConnect)
	}

	if result.ErrCode == types.ErrJWTTokenExpired {
		log.Info("JWT expired, attempting to re-login")
		if err := c.Login(ctx); err != nil {
			log.Error("JWT expired, re-login failed", "error", err)
			return fmt.Errorf("JWT expired, re-login failed: %w", ErrCoordinatorConnect)
		}
		log.Info("re-login success")
		return c.SubmitProof(ctx, req)
	}

	if result.ErrCode != types.Success {
		return fmt.Errorf("error code: %v, error message: %v", result.ErrCode, result.ErrMsg)
	}

	return nil
}
