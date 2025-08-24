package proxy

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"

	"scroll-tech/coordinator/internal/types"
)

// Client wraps an http client with a preset host for coordinator API calls
type Client struct {
	httpClient *http.Client
	host       string
	loginToken string
}

// NewClient creates a new Client with the specified host
func NewClient(host string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		host: host,
	}
}

// NewClientWithHTTPClient creates a new Client with a custom http.Client
func NewClientWithHTTPClient(host string, httpClient *http.Client) *Client {
	return &Client{
		httpClient: httpClient,
		host:       host,
	}
}

// FullLogin performs the complete login process: get challenge then login
func (c *Client) Login(param types.LoginParameter) (*types.LoginSchema, error) {
	// Step 1: Get challenge
	url := fmt.Sprintf("%s/v1/challenge", c.host)

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
	url = fmt.Sprintf("%s/v1/login", c.host)

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
func (c *Client) ProxyLogin(param types.LoginParameter) (*http.Response, error) {
	url := fmt.Sprintf("%s/v1/proxy_login", c.host)

	jsonData, err := json.Marshal(param)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proxy login parameter: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.loginToken)

	return c.httpClient.Do(req)
}

// GetTask makes a POST request to /v1/get_task with GetTaskParameter
func (c *Client) GetTask(param types.GetTaskParameter, token string) (*http.Response, error) {
	url := fmt.Sprintf("%s/v1/get_task", c.host)

	jsonData, err := json.Marshal(param)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get task parameter: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
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
func (c *Client) SubmitProof(param types.SubmitProofParameter, token string) (*http.Response, error) {
	url := fmt.Sprintf("%s/v1/submit_proof", c.host)

	jsonData, err := json.Marshal(param)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal submit proof parameter: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create submit proof request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.httpClient.Do(req)
}

// transformToValidPrivateKey safely transforms arbitrary bytes into valid private key bytes
func (c *Client) buildPrivateKey(inputBytes []byte) (*ecdsa.PrivateKey, error) {
	// Try appending bytes from 0x0 to 0x20 until we get a valid private key
	for appendByte := byte(0x0); appendByte <= 0x20; appendByte++ {
		// Append the byte to input
		extendedBytes := append(inputBytes, appendByte)

		// Calculate 256-bit hash
		hash := crypto.Keccak256(extendedBytes)

		// Try to create private key from hash
		if k, err := crypto.ToECDSA(hash); err == nil {
			return k, nil
		}
	}

	return nil, fmt.Errorf("failed to generate valid private key from input bytes")
}

func (c *Client) generateLoginParameter(privateKeyBytes []byte, challenge string) (*types.LoginParameter, error) {
	// Generate private key
	privKey, err := c.buildPrivateKey(privateKeyBytes)
	if err != nil {
		return nil, err
	}

	// Generate public key string
	publicKeyHex := common.Bytes2Hex(crypto.CompressPubkey(&privKey.PublicKey))

	// Create login parameter with proxy settings
	loginParam := &types.LoginParameter{
		Message: types.Message{
			Challenge:          challenge,
			ProverName:         "proxy",
			ProverVersion:      "proxy",
			ProverProviderType: types.ProverProviderTypeProxy,
			ProverTypes:        []types.ProverType{}, // Default empty
			VKs:                []string{},           // Default empty
		},
		PublicKey: publicKeyHex,
	}

	// Sign the message with the private key
	if err := loginParam.SignWithKey(privKey); err != nil {
		return nil, fmt.Errorf("failed to sign login parameter: %w", err)
	}

	return loginParam, nil
}
