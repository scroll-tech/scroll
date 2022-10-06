package config

import (
	"os"
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
