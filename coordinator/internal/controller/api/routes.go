package api

import (
	"context"

	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
)

// ProverAPI for provers inorder to register and submit proof
type ProverAPI interface {
	RequestToken(authMsg *message.AuthMsg) (string, error)
	Register(ctx context.Context, authMsg *message.AuthMsg) (*rpc.Subscription, error)
	SubmitProof(proof *message.ProofMsg) error
}

// RegisterAPIs register api for coordinator
func RegisterAPIs(cfg *config.Config, db *gorm.DB) []rpc.API {
	return []rpc.API{
		{
			Namespace: "prover",
			Service:   ProverAPI(NewProverController(cfg.ProverManagerConfig, db)),
			Public:    true,
		},
	}
}
