package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/types/message"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// Config loads prover configuration items.
type Config struct {
	ProverName        string             `json:"prover_name"`
	KeystorePath      string             `json:"keystore_path"`
	KeystorePassword  string             `json:"keystore_password"`
	Core              *ProverCoreConfig  `json:"core"`
	DBPath            string             `json:"db_path"`
	CoordinatorConfig *CoordinatorConfig `json:"coordinator_config"`
	L2GethConfig      *L2GethConfig      `json:"l2geth_config"`
}

// ProverCoreConfig load zk prover config.
type ProverCoreConfig struct {
	ParamsPath string            `json:"params_path"`
	ProofType  message.ProofType `json:"prove_type,omitempty"` // 0: chunk prover (default type), 1: batch prover
	DumpDir    string            `json:"dump_dir,omitempty"`
}

// CoordinatorConfig represents the configuration for the Coordinator client.
type CoordinatorConfig struct {
	BaseURL              string `json:"base_url"`
	RetryCount           int    `json:"retry_count"`
	RetryWaitTimeSec     int    `json:"retry_wait_time_sec"`
	ConnectionTimeoutSec int    `json:"connection_timeout_sec"`
}

// L2GethConfig represents the configuration for the l2geth client.
type L2GethConfig struct {
	Endpoint      string          `json:"endpoint"`
	Confirmations rpc.BlockNumber `json:"confirmations"`
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
	return cfg, nil
}
