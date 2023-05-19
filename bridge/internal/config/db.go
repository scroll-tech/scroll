package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// DBConfig db config
type DBConfig struct {
	// data source name
	DSN        string `json:"dsn"`
	DriverName string `json:"driver_name"`

	MaxOpenNum int `json:"maxOpenNum"`
	MaxIdleNum int `json:"maxIdleNum"`

	SlowSqlThreshold time.Duration `json:"slow_sql_threshold"`
	ShowSql          bool          `json:"show_sql"`
}

// NewDBConfig returns a new instance of Config.
func NewDBConfig(file string) (*DBConfig, error) {
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
