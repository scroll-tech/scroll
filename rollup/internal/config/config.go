package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/database"
	"scroll-tech/common/utils"
)

// Config load configuration items.
type Config struct {
	L1Config *L1Config        `json:"l1_config"`
	L2Config *L2Config        `json:"l2_config"`
	DBConfig *database.Config `json:"db_config"`
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

	// Override config with environment variables
	err = utils.OverrideConfigWithEnv(cfg, "SCROLL_ROLLUP")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
