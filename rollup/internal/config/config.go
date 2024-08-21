package config

import (
	"scroll-tech/common/database"

	"github.com/spf13/viper"
)

// Config load configuration items.
type Config struct {
	L1Config *L1Config        `json:"l1_config"`
	L2Config *L2Config        `json:"l2_config"`
	DBConfig *database.Config `json:"db_config"`
}

// NewConfig returns a new instance of Config.
func NewConfig(file string) (*Config, error) {
	viper.SetConfigFile(file)
	viper.SetEnvPrefix("SCROLL_ROLLUP")
	viper.AutomaticEnv()
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
