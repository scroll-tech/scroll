package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"scroll-tech/common/libzkp"

	"github.com/scroll-tech/go-ethereum/log"
)

// Config loads roller configuration items.
type Config struct {
	RollerName       string               `json:"roller_name"`
	KeystorePath     string               `json:"keystore_path"`
	KeystorePassword string               `json:"keystore_password"`
	CoordinatorURL   string               `json:"coordinator_url"`
	Prover           *libzkp.ProverConfig `json:"prover"`
	DBPath           string               `json:"db_path"`
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
