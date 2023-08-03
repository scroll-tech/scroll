package types

import "time"

const (
	// PublicKey the public key for context
	PublicKey = "public_key"
	// ProverName the prover name key for context
	ProverName = "prover_name"
)

// Message the login message struct
type Message struct {
	Challenge  string `form:"challenge" json:"challenge" binding:"required"`
	ProverName string `form:"prover_name" json:"prover_name" binding:"required"`
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
