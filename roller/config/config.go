package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/types/message"

	"github.com/scroll-tech/go-ethereum/log"
)

// Config loads roller configuration items.
type Config struct {
	RollerName       string        `json:"roller_name"`
	KeystorePath     string        `json:"keystore_path"`
	KeystorePassword string        `json:"keystore_password"`
	CoordinatorURL   string        `json:"coordinator_url"`
	TraceEndpoint    string        `json:"trace_endpoint"`
	Prover           *ProverConfig `json:"prover"`
	DBPath           string        `json:"db_path"`
}

// ProverConfig load zk prover config.
type ProverConfig struct {
	ParamsPath string            `json:"params_path"`
	SeedPath   string            `json:"seed_path"`
	ProofType  message.ProofType `json:"prove_type,omitempty"` // 0: basic roller (default type), 1: aggregator roller
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
