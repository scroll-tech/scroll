package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/db"
)

// Config load configuration items.
type Config struct {
	L1Config *L1Config  `json:"l1_config"`
	L2Config *L2Config  `json:"l2_config"`
	DBConfig *db.Config `json:"db_config"`
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
