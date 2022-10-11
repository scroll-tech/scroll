package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DBConfig db config
type DBConfig struct {
	// data source name
	DSN        string `json:"dsn"`
	DriverName string `json:"driver_name"`

	MaxOpenNum int `json:"maxOpenNum" default:"200"`
	MaxIdleNUm int `json:"maxIdleNum" default:"20"`
}

// GetEnvWithDefault get value from env if is none use the default
func GetEnvWithDefault(key string, defult string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		val = defult
	}
	return val
}

// NewConfig returns a new instance of DBConfig.
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
	cfg.DSN = GetEnvWithDefault("DB_DSN", cfg.DSN)
	cfg.DriverName = GetEnvWithDefault("DB_DRIVER", cfg.DriverName)

	return cfg, nil
}
