package types

import "time"

const (
	PublicKeyCtxKey  = "public_key_context"
	ProverNameCtxKey = "prover_name_context"
)

// LoginParameter for /login api
type LoginParameter struct {
	//PublicKey  string `form:"public_key" json:"public_key" binding:"required"`
	ProverName string `form:"prover_name" json:"prover_name" binding:"required"`
}

// LoginSchema for /login response
type LoginSchema struct {
	Time  time.Time `json:"time"`
	Token string    `json:"token"`
}
