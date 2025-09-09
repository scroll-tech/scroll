package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/utils"
)

// Proxy loads proxy configuration items.
type ProxyManager struct {
	// Zk verifier config help to confine the connected prover.
	Verifier *VerifierConfig `json:"verifier"`
	Client   *ProxyClient    `json:"proxy_cli"`
	Auth     *Auth           `json:"auth"`
}

func (m *ProxyManager) Normalize() {
	if m.Client.Secret == "" {
		m.Client.Secret = m.Auth.Secret
	}

	if m.Client.ProxyVersion == "" {
		m.Client.ProxyVersion = m.Verifier.MinProverVersion
	}
}

// Proxy client configuration for connect to upstream as a client
type ProxyClient struct {
	ProxyName    string `json:"proxy_name"`
	ProxyVersion string `json:"proxy_version,omitempty"`
	Secret       string `json:"secret,omitempty"`
}

// Coordinator configuration
type UpStream struct {
	BaseUrl              string `json:"base_url"`
	RetryCount           uint   `json:"retry_count"`
	RetryWaitTime        uint   `json:"retry_wait_time_sec"`
	ConnectionTimeoutSec uint   `json:"connection_timeout_sec"`
}

// Config load configuration items.
type ProxyConfig struct {
	ProxyManager *ProxyManager        `json:"proxy_manager"`
	ProxyName    string               `json:"proxy_name"`
	Coordinators map[string]*UpStream `json:"coondiators"`
}

// NewConfig returns a new instance of Config.
func NewProxyConfig(file string) (*ProxyConfig, error) {
	buf, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	cfg := &ProxyConfig{}
	err = json.Unmarshal(buf, cfg)
	if err != nil {
		return nil, err
	}

	// Override config with environment variables
	err = utils.OverrideConfigWithEnv(cfg, "SCROLL_COORDINATOR_PROXY")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
