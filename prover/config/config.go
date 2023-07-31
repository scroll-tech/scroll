package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/scroll-tech/go-ethereum/log"
)

// Config loads prover configuration items.
type Config struct {
	ProverName       string             `json:"prover_name"`
	KeystorePath     string             `json:"keystore_path"`
	KeystorePassword string             `json:"keystore_password"`
	CoordinatorURL   string             `json:"coordinator_url"`
	TraceEndpoint    string             `json:"trace_endpoint"`
	DBPath           string             `json:"db_path"`
	BatchConfig      *BatchProverConfig `json:"batch_config"`
	ChunkConfig      *ChunkProverConfig `json:"chunk_config"`
}

// BatchProverConfig load batch prover config.
type BatchProverConfig struct {
	ParamsPath string `json:"params_path"`
	AssetsPath string `json:"assets_path"`
	DumpDir    string `json:"dump_dir,omitempty"`
}

// ChunkProverConfig load chunk prover config.
type ChunkProverConfig struct {
	ParamsPath string `json:"params_path"`
	DumpDir    string `json:"dump_dir,omitempty"`
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
