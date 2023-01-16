package database

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// RedisConfig redis cache config.
type RedisConfig struct {
	RedisURL       string `json:"redis_url"`
	TraceExpireSec int64  `json:"trace_expire_sec"`
}

// DBConfig db config
type DBConfig struct {
	// data source name
	DSN        string `json:"dsn"`
	DriverName string `json:"driver_name"`

	MaxOpenNum int `json:"maxOpenNum" default:"200"`
	MaxIdleNum int `json:"maxIdleNum" default:"20"`

	// Redis config
	RedisConfig *RedisConfig `json:"redis_config"`
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
