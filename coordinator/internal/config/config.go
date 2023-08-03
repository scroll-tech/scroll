package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"scroll-tech/common/database"
)

const (
	defaultNumberOfVerifierWorkers      = 10
	defaultNumberOfSessionRetryAttempts = 2
)

// ProverManagerConfig loads sequencer configuration items.
type ProverManagerConfig struct {
	CompressionLevel int `json:"compression_level,omitempty"`
	// asc or desc (default: asc)
	OrderSession string `json:"order_session,omitempty"`
	// The amount of provers to pick per proof generation session.
	ProversPerSession uint8 `json:"provers_per_session"`
	// Number of attempts that a session can be retried if previous attempts failed.
	// Currently we only consider proving timeout as failure here.
	SessionAttempts uint8 `json:"session_attempts,omitempty"`
	// Zk verifier config.
	Verifier *VerifierConfig `json:"verifier,omitempty"`
	// Proof collection time (in seconds).
	CollectionTime int `json:"collection_time"`
	// Token time to live (in seconds)
	TokenTimeToLive int `json:"token_time_to_live"`
	// Max number of workers in verifier worker pool
	MaxVerifierWorkers int `json:"max_verifier_workers,omitempty"`
}

// L2Config loads l2geth configuration items.
type L2Config struct {
	// l2geth chain_id.
	ChainID uint64 `json:"chain_id"`
}

// Auth provides the auth of prover-stats-api
type Auth struct {
	Secret                  string `json:"secret"`
	ChallengeExpireDuration int    `json:"challenge_expire_duration"`
	LoginExpireDuration     int    `json:"token_expire_duration"` // unit: seconds
}

// Config load configuration items.
type Config struct {
	ProverManagerConfig *ProverManagerConfig `json:"prover_manager_config"`
	DBConfig            *database.Config     `json:"db_config"`
	L2Config            *L2Config            `json:"l2_config"`
	Auth                *Auth                `json:"auth"`
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

	// Check prover's order session
	order := strings.ToUpper(cfg.ProverManagerConfig.OrderSession)
	if len(order) > 0 && !(order == "ASC" || order == "DESC") {
		return nil, errors.New("prover config's order session is invalid")
	}
	cfg.ProverManagerConfig.OrderSession = order

	if cfg.ProverManagerConfig.MaxVerifierWorkers == 0 {
		cfg.ProverManagerConfig.MaxVerifierWorkers = defaultNumberOfVerifierWorkers
	}
	if cfg.ProverManagerConfig.SessionAttempts == 0 {
		cfg.ProverManagerConfig.SessionAttempts = defaultNumberOfSessionRetryAttempts
	}

	return cfg, nil
}
