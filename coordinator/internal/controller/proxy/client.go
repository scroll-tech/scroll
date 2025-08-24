package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/types"
)

type ClientHelper interface {
	GenLoginParam(string) (*types.LoginParameter, error)
	OnError(isUnauth bool)
}

// Client wraps an http client with a preset host for coordinator API calls
type upClient struct {
	httpClient *http.Client
	baseURL    string
	loginToken string
	helper     ClientHelper
}

// NewClient creates a new Client with the specified host
func newUpClient(cfg *config.UpStream, helper ClientHelper) *upClient {
	return &upClient{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.ConnectionTimeoutSec) * time.Second,
		},
		baseURL: cfg.BaseUrl,
		helper:  helper,
	}
}

// FullLogin performs the complete login process: get challenge then login
func (c *upClient) Login(ctx context.Context) (*types.LoginSchema, error) {
	// Step 1: Get challenge
	url := fmt.Sprintf("%s/coordinator/v1/challenge", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create challenge request: %w", err)
	}

	challengeResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}
	defer challengeResp.Body.Close()

	if challengeResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("challenge request failed with status: %d", challengeResp.StatusCode)
	}

	// Step 2: Parse challenge response
	var loginSchema types.LoginSchema
	if err := json.NewDecoder(challengeResp.Body).Decode(&loginSchema); err != nil {
		return nil, fmt.Errorf("failed to parse challenge response: %w", err)
	}

	// Step 3: Use the token from challenge as Bearer token for login
	url = fmt.Sprintf("%s/coordinator/v1/login", c.baseURL)

	param, err := c.helper.GenLoginParam(loginSchema.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to setup login parameter: %w", err)
	}

	jsonData, err := json.Marshal(param)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login parameter: %w", err)
	}

	req, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+loginSchema.Token)

	loginResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform login request: %w", err)
	}

	// Parse login response as LoginSchema and store the token
	if loginResp.StatusCode == http.StatusOK {
		var loginResult types.LoginSchema
		if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err == nil {
			c.loginToken = loginResult.Token
		}
		// Note: Body is consumed after decoding, caller should not read it again
		return &loginResult, nil
	}

	return nil, fmt.Errorf("login request failed with status: %d", loginResp.StatusCode)
}

// ProxyLogin makes a POST request to /v1/proxy_login with LoginParameter
func (c *upClient) ProxyLogin(ctx *gin.Context, param types.LoginParameter) (*http.Response, error) {
	url := fmt.Sprintf("%s/coordinator/v1/proxy_login", c.baseURL)

	jsonData, err := json.Marshal(param)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proxy login parameter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.loginToken)

	return c.httpClient.Do(req)
}

// GetTask makes a POST request to /v1/get_task with GetTaskParameter
func (c *upClient) GetTask(ctx *gin.Context, param types.GetTaskParameter, token string) (*http.Response, error) {
	url := fmt.Sprintf("%s/coordinator/v1/get_task", c.baseURL)

	jsonData, err := json.Marshal(param)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get task parameter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create get task request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.httpClient.Do(req)
}

// SubmitProof makes a POST request to /v1/submit_proof with SubmitProofParameter
func (c *upClient) SubmitProof(ctx *gin.Context, param types.SubmitProofParameter, token string) (*http.Response, error) {
	url := fmt.Sprintf("%s/coordinator/v1/submit_proof", c.baseURL)

	jsonData, err := json.Marshal(param)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal submit proof parameter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create submit proof request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.httpClient.Do(req)
}
