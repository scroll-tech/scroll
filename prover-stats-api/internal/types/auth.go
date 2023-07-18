package types

import "time"

type LoginParameter struct {
	PublicKey string `form:"public_key" json:"public_key" binding:"required"`
}

type LoginSchema struct {
	Time  time.Time `json:"time"`
	Token string    `json:"token"`
}
