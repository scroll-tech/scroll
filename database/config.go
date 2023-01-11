package database

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// RedisConfig redis cache config.
type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password,omitempty"`
}

// DBConfig db config
type DBConfig struct {
	// data source name
	DSN        string `json:"dsn"`
	DriverName string `json:"driver_name"`

	MaxOpenNum int `json:"maxOpenNum" default:"200"`
	MaxIdleNum int `json:"maxIdleNum" default:"20"`

	// Redis config
	RedisConfig *RedisConfig `json:"redis_config,omitempty"`
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
