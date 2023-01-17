package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/database"
)

// Config load configuration items.
type Config struct {
	L1Config *L1Config          `json:"l1_config"`
	L2Config *L2Config          `json:"l2_config"`
	DBConfig *database.DBConfig `json:"db"`
}

// NewConfig returns a new instance of Config.
func NewConfig(file string) (*Config, error) {
	buf, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = json.Unmarshal(buf, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
