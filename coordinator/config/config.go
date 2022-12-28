package config

import (
	"scroll-tech/common/viper"
)

// NewConfig returns a new instance of Config.
func NewConfig(file string) error {
	viper.SetConfigFile(file)
	return viper.ReadInConfig()
}
