package database

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/utils"
)

// DBConfig db config
type DBConfig struct {
	// data source name
	DSN        string `json:"dsn"`
	DriverName string `json:"driver_name"`

	MaxOpenNum int `json:"maxOpenNum" default:"200"`
	MaxIdleNum int `json:"maxIdleNum" default:"20"`
}

// NewConfig returns a new instance of Config.
func NewConfig(file string) (*DBConfig, error) {
	buf, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	cfg := &DBConfig{}
	err = json.Unmarshal(buf, cfg)
	if err != nil {
		return nil, err
	}

	// cover value by env fields
	cfg.DSN = utils.GetEnvWithDefault("DB_DSN", cfg.DSN)
	cfg.DriverName = utils.GetEnvWithDefault("DB_DRIVER", cfg.DriverName)

	return cfg, nil
}
