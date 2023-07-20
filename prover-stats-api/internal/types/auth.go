package types

import "time"

// LoginParameter for /login api
type LoginParameter struct {
	PublicKey string `form:"public_key" json:"public_key" binding:"required"`
}

// LoginSchema for /login response
type LoginSchema struct {
	Time  time.Time `json:"time"`
	Token string    `json:"token"`
}
