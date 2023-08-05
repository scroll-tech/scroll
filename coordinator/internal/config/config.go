package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/database"
)

const (
	defaultNumberOfVerifierWorkers      = 10
	defaultNumberOfSessionRetryAttempts = 2
)

// ProverManager loads sequencer configuration items.
type ProverManager struct {
	// The amount of provers to pick per proof generation session.
	ProversPerSession uint8 `json:"provers_per_session"`
	// Number of attempts that a session can be retried if previous attempts failed.
	// Currently we only consider proving timeout as failure here.
	SessionAttempts uint8 `json:"session_attempts,omitempty"`
	// Zk verifier config.
	Verifier *VerifierConfig `json:"verifier,omitempty"`
	// Proof collection time (in seconds).
	CollectionTimeSec int `json:"collection_time_sec"`
	// Max number of workers in verifier worker pool
	MaxVerifierWorkers int `json:"max_verifier_workers,omitempty"`
}

// L2 loads l2geth configuration items.
type L2 struct {
	// l2geth chain_id.
	ChainID uint64 `json:"chain_id"`
}

// Auth provides the auth of prover-stats-api
type Auth struct {
	Secret                     string `json:"secret"`
	ChallengeExpireDurationSec int    `json:"challenge_expire_duration_sec"`
	LoginExpireDurationSec     int    `json:"token_expire_duration_sec"` // unit: seconds
}

// Config load configuration items.
type Config struct {
	ProverManager *ProverManager   `json:"prover_manager"`
	DB            *database.Config `json:"db"`
	L2            *L2              `json:"l2"`
	Auth          *Auth            `json:"auth"`
}

// VerifierConfig load zk verifier config.
type VerifierConfig struct {
	MockMode   bool   `json:"mock_mode"`
	ParamsPath string `json:"params_path"`
	AssetsPath string `json:"assets_path"`
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

	if cfg.ProverManager.MaxVerifierWorkers == 0 {
		cfg.ProverManager.MaxVerifierWorkers = defaultNumberOfVerifierWorkers
	}
	if cfg.ProverManager.SessionAttempts == 0 {
		cfg.ProverManager.SessionAttempts = defaultNumberOfSessionRetryAttempts
	}

	return cfg, nil
}
