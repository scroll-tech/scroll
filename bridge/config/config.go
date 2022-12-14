package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/utils"

	"scroll-tech/database"
)

// Config load configuration items.
type Config struct {
	L1Config *L1Config          `json:"l1_config"`
	L2Config *L2Config          `json:"l2_config"`
	DBConfig *database.DBConfig `json:"db_config"`
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

	// cover value by env fields
	cfg.DBConfig.DSN = utils.GetEnvWithDefault("DB_DSN", cfg.DBConfig.DSN)
	cfg.DBConfig.DriverName = utils.GetEnvWithDefault("DB_DRIVER", cfg.DBConfig.DriverName)

	return cfg, nil
}
