package api

import (
	"context"

	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
)

// RollerAPI for rollers inorder to register and submit proof
type RollerAPI interface {
	RequestToken(authMsg *message.AuthMsg) (string, error)
	Register(ctx context.Context, authMsg *message.AuthMsg) (*rpc.Subscription, error)
	SubmitProof(proof *message.ProofMsg) error
}

// RollerDebugAPI roller api interface in order go get debug message.
type RollerDebugAPI interface {
	// ListRollers returns all live rollers
	//ListRollers() ([]*RollerInfo, error)
	//// GetSessionInfo returns the session information given the session id.
	//GetSessionInfo(sessionID string) (*SessionInfo, error)
}

// APIs register api for coordinator
func APIs(cfg *config.Config, db *gorm.DB) []rpc.API {
	return []rpc.API{
		{
			Namespace: "roller",
			Service:   RollerAPI(NewRoller(cfg, db)),
			Public:    true,
		},
		//{
		//	Namespace: "debug",
		//	Public:    true,
		//	Service:   RollerDebugAPI(NewRollerDebug()),
		//},
	}
}
