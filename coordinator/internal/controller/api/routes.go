package api

import (
	"context"

	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
)

// RollerAPI for provers inorder to register and submit proof
type RollerAPI interface {
	RequestToken(authMsg *message.AuthMsg) (string, error)
	Register(ctx context.Context, authMsg *message.AuthMsg) (*rpc.Subscription, error)
	SubmitProof(proof *message.ProofMsg) error
}

// RegisterAPIs register api for coordinator
func RegisterAPIs(cfg *config.Config, db *gorm.DB) []rpc.API {
	return []rpc.API{
		{
			Namespace: "prover",
			Service:   RollerAPI(NewRollerController(cfg.RollerManagerConfig, db)),
			Public:    true,
		},
	}
}
