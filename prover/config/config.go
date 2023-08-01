package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"scroll-tech/common/types/message"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// Config loads prover configuration items.
type Config struct {
	ProverName       string             `json:"prover_name"`
	KeystorePath     string             `json:"keystore_path"`
	KeystorePassword string             `json:"keystore_password"`
	TraceEndpoint    string             `json:"trace_endpoint"`
	Core             *ProverCoreConfig  `json:"core"`
	DBPath           string             `json:"db_path"`
	Coordinator      *CoordinatorConfig `json:"coordinator"`
	Confirmations    rpc.BlockNumber    `json:"confirmations"`
}

// ProverCoreConfig load zk prover config.
type ProverCoreConfig struct {
	ParamsPath string            `json:"params_path"`
	ProofType  message.ProofType `json:"prove_type,omitempty"` // 0: chunk prover (default type), 1: batch prover
	DumpDir    string            `json:"dump_dir,omitempty"`
}

// CoordinatorConfig represents the configuration for the Coordinator client.
type CoordinatorConfig struct {
	Timeout       int    `json:"timeout"`
	BaseURL       string `json:"base_url"`
	RetryCount    int    `json:"retry_count"`
	RetryWaitTime int    `json:"retry_wait_time"`
}

// NewConfig returns a new instance of Config.
func NewConfig(file string) (*Config, error) {
	buf, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err = json.Unmarshal(buf, cfg); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(cfg.DBPath) {
		if cfg.DBPath, err = filepath.Abs(cfg.DBPath); err != nil {
			log.Error("Failed to get abs path", "error", err)
			return nil, err
		}
	}
	if cfg.Coordinator == nil || cfg.Coordinator.BaseURL == "" {
		return nil, errors.New("missing coordinator config or base_url in configuration")
	}
	return cfg, nil
}
