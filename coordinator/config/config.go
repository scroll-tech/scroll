package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"scroll-tech/common/utils"
	db_config "scroll-tech/database"
)

// RollerManagerConfig loads sequencer configuration items.
type RollerManagerConfig struct {
	// Endpoint to set websocket server up on.
	Endpoint string `json:"endpoint"`
	// asc or desc (default: asc)
	OrderSession string `json:"order_session,omitempty"`
	// The amount of rollers to pick per proof generation session.
	RollersPerSession uint8 `json:"rollers_per_session"`
	// Unix socket endpoint to which we send proofs to be verified.
	VerifierEndpoint string `json:"verifier_endpoint,omitempty"`
	// Proof collection time (in minutes).
	CollectionTime int `json:"collection_time"`
}

// Config load configuration items.
type Config struct {
	RollerManagerConfig *RollerManagerConfig `json:"roller_manager_config"`
	DBConfig            *db_config.DBConfig  `json:"db_config"`
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

	// Check roller's order session
	order := strings.ToUpper(cfg.RollerManagerConfig.OrderSession)
	if len(order) > 0 && !(order == "ASC" || order == "DESC") {
		return nil, errors.New("roller config's order session is invalid")
	}
	cfg.RollerManagerConfig.OrderSession = order

	return cfg, nil
}
