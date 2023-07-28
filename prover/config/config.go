package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/types/message"

	"github.com/scroll-tech/go-ethereum/log"
)

// Config loads prover configuration items.
type Config struct {
	ProverName       string            `json:"prover_name"`
	KeystorePath     string            `json:"keystore_path"`
	KeystorePassword string            `json:"keystore_password"`
	CoordinatorURL   string            `json:"coordinator_url"`
	TraceEndpoint    string            `json:"trace_endpoint"`
	Core             *ProverCoreConfig `json:"core"`
	DBPath           string            `json:"db_path"`
}

// ProverCoreConfig load zk prover config.
type ProverCoreConfig struct {
	ParamsPath string            `json:"params_path"`
	SeedPath   string            `json:"seed_path"`
	ProofType  message.ProofType `json:"prove_type,omitempty"` // 0: chunk prover (default type), 1: batch prover
	DumpDir    string            `json:"dump_dir,omitempty"`
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
