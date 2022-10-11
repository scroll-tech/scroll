package config

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/scroll-tech/go-ethereum/log"
)

// Config loads roller configuration items.
type Config struct {
	RollerName       string        `toml:"roller_name"`
	KeystorePath     string        `toml:"keystore_path"`
	KeystorePassword string        `toml:"keystore_password"`
	ScrollURL        string        `toml:"scroll_url"`
	Prover           *ProverConfig `toml:"prover"`
	DBPath           string        `toml:"db_path"`
}

// ProverConfig loads zk roller configuration items.
type ProverConfig struct {
	MockMode   bool   `toml:"mock_mode"`
	ParamsPath string `toml:"params_path"`
	SeedPath   string `toml:"seed_path"`
}

// InitConfig inits config from file.
func InitConfig(path string) (*Config, error) {
	cfg := &Config{}
	_, err := toml.DecodeFile(path, cfg)
	if err != nil {
		log.Error("init config failed", "error", err)
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
