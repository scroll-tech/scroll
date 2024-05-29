package types

import "time"

const (
	// PublicKey the public key for context
	PublicKey = "public_key"
	// ProverName the prover name key for context
	ProverName = "prover_name"
	// ProverVersion the prover version for context
	ProverVersion = "prover_version"
	// HardForkName the fork name for context
	HardForkName = "hard_fork_name"
)

// Message the login message struct
type Message struct {
	Challenge     string `form:"challenge" json:"challenge" binding:"required"`
	ProverVersion string `form:"prover_version" json:"prover_version" binding:"required"`
	ProverName    string `form:"prover_name" json:"prover_name" binding:"required"`
	HardForkName  string `form:"hard_fork_name" json:"hard_fork_name"`
}

// LoginParameter for /login api
type LoginParameter struct {
	Message   Message `form:"message" json:"message" binding:"required"`
	Signature string  `form:"signature" json:"signature" binding:"required"`
}

// LoginSchema for /login response
type LoginSchema struct {
	Time  time.Time `json:"time"`
	Token string    `json:"token"`
}
