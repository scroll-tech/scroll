package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/database"
)

// Config provides the config of prover-stats-api
type Config struct {
	DBConfig *database.Config `json:"db_config"`
	Auth     Auth             `json:"auth"`
}

// Auth provides the auth of prover-stats-api
type Auth struct {
	Secret              string `json:"secret"`
	TokenExpireDuration int    `json:"token_expire_duration"` // unit: seconds
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
