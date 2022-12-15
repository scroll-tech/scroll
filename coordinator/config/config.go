package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/utils"

	apollo_config "scroll-tech/common/apollo"
	db_config "scroll-tech/database"
)

// RollerManagerConfig loads sequencer configuration items.
type RollerManagerConfig struct {
	// Zk verifier config.
	Verifier *VerifierConfig `json:"verifier,omitempty"`
}

// L2Config loads l2geth configuration items.
type L2Config struct {
	// l2geth node url.
	Endpoint string `json:"endpoint"`
}

// Config load configuration items.
type Config struct {
	RollerManagerConfig *RollerManagerConfig `json:"roller_manager_config"`
	DBConfig            *db_config.DBConfig  `json:"db_config"`
	L2Config            *L2Config            `json:"l2_config"`
}

// VerifierConfig load zk verifier config.
type VerifierConfig struct {
	MockMode   bool   `json:"mock_mode"`
	ParamsPath string `json:"params_path"`
	AggVkPath  string `json:"agg_vk_path"`
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

	// cover value by env fields
	cfg.DBConfig.DSN = utils.GetEnvWithDefault("DB_DSN", cfg.DBConfig.DSN)
	cfg.DBConfig.DriverName = utils.GetEnvWithDefault("DB_DRIVER", cfg.DBConfig.DriverName)

	return cfg, nil
}

// GetOrderSession : get session order, asc or desc (default: asc).
func GetOrderSession() string {
	return apollo_config.AgolloClient.GetStringValue("orderSession", "ASC")
}

// GetCollectionTime : get proof collection time (in minutes).
func GetCollectionTime() int {
	return apollo_config.AgolloClient.GetIntValue("collectionTime", 1)
}

// GetTokenTimeToLive : get token time to live (in seconds).
func GetTokenTimeToLive() int {
	return apollo_config.AgolloClient.GetIntValue("tokenTimeToLive", 5)
}

// GetProofAndPkBufferSize : get proof and pk buffer size.
func GetProofAndPkBufferSize() int {
	return apollo_config.AgolloClient.GetIntValue("proofAndPkBufferSize", 10)
}
