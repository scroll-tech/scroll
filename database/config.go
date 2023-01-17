package database

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/database/cache"
)

// PGConfig postgres db config.
type PGConfig struct {
	// data source name
	DSN        string `json:"dsn"`
	DriverName string `json:"driver_name"`

	MaxOpenNum int `json:"maxOpenNum" default:"200"`
	MaxIdleNum int `json:"maxIdleNum" default:"20"`
}

// DBConfig db config
type DBConfig struct {
	DB          *PGConfig          `json:"persistence"`
	RedisConfig *cache.RedisConfig `json:"redis,omitempty"`
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

	return cfg, nil
}
