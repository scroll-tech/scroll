package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"scroll-tech/common/database"
)

type Config struct {
	DBConfig  *database.Config `json:"db_config"`
	ApiSecret string           `json:"api_secret"`
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
