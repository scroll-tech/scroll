//nolint:errcheck,bodyclose // body is closed in the following handleHttpResp call
package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"

	ctypes "scroll-tech/common/types"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/types"
)

// Client wraps an http client with a preset host for coordinator API calls
type upClient struct {
	httpClient      *http.Client
	baseURL         string
	loginToken      string
	compatibileMode bool
}

// NewClient creates a new Client with the specified host
func newUpClient(cfg *config.UpStream) *upClient {
	return &upClient{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.ConnectionTimeoutSec) * time.Second,
		},
		baseURL:         cfg.BaseUrl,
		compatibileMode: cfg.CompatibileMode,
	}
}

func (c *upClient) Token() string {
	return c.loginToken
}

// need a parsable schema definition
type loginSchema struct {
	Time  string `json:"time"`
	Token string `json:"token"`
}

// Login performs the complete login process: get challenge then login
func (c *upClient) Login(ctx context.Context, genLogin func(string) (*types.LoginParameter, error)) (*ctypes.Response, error) {
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

	parsedResp, err := handleHttpResp(challengeResp)
	if err != nil {
		return nil, err
	} else if parsedResp.ErrCode != 0 {
		return nil, fmt.Errorf("challenge failed: %d (%s)", parsedResp.ErrCode, parsedResp.ErrMsg)
	}

	// Ste p2: Parse challenge response
	var challengeSchema loginSchema
	if err := parsedResp.DecodeData(&challengeSchema); err != nil {
		return nil, fmt.Errorf("failed to parse challenge response: %w", err)
	}

	// Step 3: Use the token from challenge as Bearer token for login
	url = fmt.Sprintf("%s/coordinator/v1/login", c.baseURL)

	param, err := genLogin(challengeSchema.Token)
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
	req.Header.Set("Authorization", "Bearer "+challengeSchema.Token)

	loginResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform login request: %w", err)
	}
	return handleHttpResp(loginResp)
}

func handleHttpResp(resp *http.Response) (*ctypes.Response, error) {
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		defer resp.Body.Close()
		var respWithData ctypes.Response
		// Note: Body is consumed after decoding, caller should not read it again
		if err := json.NewDecoder(resp.Body).Decode(&respWithData); err == nil {
			return &respWithData, nil
		} else {
			return nil, fmt.Errorf("login parsing expected response failed: %v", err)
		}

	}
	return nil, fmt.Errorf("login request failed with status: %d", resp.StatusCode)
}

func (c *upClient) proxyLoginCompatibleMode(ctx context.Context, param *types.LoginParameter) (*ctypes.Response, error) {
	mimePrivK, err := buildPrivateKey([]byte(param.PublicKey))
	if err != nil {
		return nil, err
	}
	mimePkHex := common.Bytes2Hex(crypto.CompressPubkey(&mimePrivK.PublicKey))

	genLoginParam := func(challenge string) (*types.LoginParameter, error) {

		// Create login parameter with proxy settings
		loginParam := &types.LoginParameter{
			Message:   param.Message,
			PublicKey: mimePkHex,
		}
		loginParam.Message.Challenge = challenge

		// Sign the message with the private key
		if err := loginParam.SignWithKey(mimePrivK); err != nil {
			return nil, fmt.Errorf("failed to sign login parameter: %w", err)
		}

		return loginParam, nil
	}

	return c.Login(ctx, genLoginParam)
}

// ProxyLogin makes a POST request to /v1/proxy_login with LoginParameter
func (c *upClient) ProxyLogin(ctx context.Context, param *types.LoginParameter) (*ctypes.Response, error) {

	if c.compatibileMode {
		return c.proxyLoginCompatibleMode(ctx, param)
	}

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

	proxyLoginResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform proxy login request: %w", err)
	}
	return handleHttpResp(proxyLoginResp)
}

// GetTask makes a POST request to /v1/get_task with GetTaskParameter
func (c *upClient) GetTask(ctx context.Context, param *types.GetTaskParameter, token string) (*ctypes.Response, error) {
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return handleHttpResp(resp)
}

// SubmitProof makes a POST request to /v1/submit_proof with SubmitProofParameter
func (c *upClient) SubmitProof(ctx context.Context, param *types.SubmitProofParameter, token string) (*ctypes.Response, error) {
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return handleHttpResp(resp)
}
