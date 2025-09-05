package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	ctypes "scroll-tech/common/types"
	"scroll-tech/coordinator/internal/types"
)

// Client wraps an http client with a preset host for coordinator API calls
type proverSession struct {
	sync.RWMutex
	proverToken string
}

func (c *proverSession) doProverLogin(ctx context.Context, cliMgr Client, param *types.LoginParameter) (*types.LoginSchema, error) {
	cli := cliMgr.Client(ctx)
	if cli == nil {
		return nil, fmt.Errorf("get upstream cli fail")
	}

	// like SDK, we would try one more time if the upstream token is expired
	resp, err := cli.ProxyLogin(ctx, param)
	if err != nil {
		return nil, fmt.Errorf("proxylogin fail: %v", err)
	}

	if resp.ErrCode == ctypes.ErrJWTTokenExpired {
		cliMgr.Reset(cli)
		cli = cliMgr.Client(ctx)
		if cli == nil {
			return nil, fmt.Errorf("get upstream cli fail (secondary try)")
		}

		// like SDK, we would try one more time if the upstream token is expired
		resp, err = cli.ProxyLogin(ctx, param)
		if err != nil {
			return nil, fmt.Errorf("proxylogin fail: %v", err)
		}
	}

	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("upstream fail: %d (%s)", resp.ErrCode, resp.ErrMsg)
	}

	var loginResult types.LoginSchema
	if err := resp.DecodeData(&loginResult); err != nil {
		return nil, err
	}

	return &loginResult, nil
}

// ProxyLogin makes a POST request to /v1/proxy_login with LoginParameter
func (c *proverSession) ProxyLogin(ctx context.Context, cli Client, param *types.LoginParameter) (*types.LoginSchema, error) {
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
	defer proxyLoginResp.Body.Close()

	// Call helper's OnResp method with the response
	c.helper.OnResp(c, proxyLoginResp)

	// Parse proxy login response as LoginSchema
	if proxyLoginResp.StatusCode == http.StatusOK {
		var loginResult types.LoginSchema
		if err := json.NewDecoder(proxyLoginResp.Body).Decode(&loginResult); err == nil {
			return &loginResult, nil
		}
		// If parsing fails, still return success but with nil result
		return nil, nil
	}

	return nil, fmt.Errorf("proxy login request failed with status: %d", proxyLoginResp.StatusCode)
}

// GetTask makes a POST request to /v1/get_task with GetTaskParameter
func (c *proverSession) GetTask(ctx context.Context, param *types.GetTaskParameter, token string) (*http.Response, error) {
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
func (c *proverSession) SubmitProof(ctx context.Context, param *types.SubmitProofParameter, token string) (*http.Response, error) {
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
