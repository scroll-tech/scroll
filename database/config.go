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

	MaxOpenNum int `json:"maxOpenNum"`
	MaxIdleNum int `json:"maxIdleNum"`

	// UseIAMAuth authenticates to AWS RDS/Aurora with short-lived IAM tokens
	// instead of a DSN password. The DSN should omit the password and enable TLS.
	UseIAMAuth bool `json:"useIAMAuth"`
	// AWSRegion signs the IAM tokens. Optional; falls back to the default AWS
	// config chain (e.g. AWS_REGION) when empty.
	AWSRegion string `json:"awsRegion"`
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

	// Override config with environment variables
	err = utils.OverrideConfigWithEnv(cfg, "SCROLL_ROLLUP_DB_CONFIG")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
