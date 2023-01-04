package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/scroll-tech/go-ethereum/common"

	"scroll-tech/common/docker"
	"scroll-tech/database"
)

// Config load configuration items.
type Config struct {
	L1Config *L1Config          `json:"l1_config"`
	L2Config *L2Config          `json:"l2_config"`
	DBConfig *database.DBConfig `json:"db_config"`
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

	return cfg, nil
}

// SetDeployedContract covers contract address in config file to actual address in docker.AddressFile
func SetDeployedContract(addressfile docker.AddressFile, config *Config) {
	config.L1Config.L1MessengerAddress = common.HexToAddress(addressfile.L1.L1ScrollMessenger.Implementation)
	config.L1Config.RelayerConfig.MessengerContractAddress = common.HexToAddress(addressfile.L2.L2ScrollMessenger)
	config.L1Config.RelayerConfig.RollupContractAddress = common.HexToAddress(addressfile.L1.ZKRollup.Implementation)
	config.L2Config.L2MessengerAddress = common.HexToAddress(addressfile.L2.L2ScrollMessenger)
	config.L2Config.RelayerConfig.MessengerContractAddress = common.HexToAddress(addressfile.L1.L1ScrollMessenger.Implementation)
	config.L2Config.RelayerConfig.RollupContractAddress = common.HexToAddress(addressfile.L1.ZKRollup.Implementation)
}
