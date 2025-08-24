package proxy

import (
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/verifier"
)

var (
	// GetTask the prover task controller
	GetTask *GetTaskController
	// SubmitProof the submit proof controller
	SubmitProof *SubmitProofController
	// Auth the auth controller
	Auth *AuthController
)

// Clients manager a series of thread-safe clients for requesting upstream
// coordinators
type Clients map[string]*Client

// InitController inits Controller with database
func InitController(cfg *config.ProxyConfig) {
	vf, err := verifier.NewVerifier(cfg.ProxyManager.Verifier)
	if err != nil {
		panic("proof receiver new verifier failure")
	}

	log.Info("verifier created", "openVmVerifier", vf.OpenVMVkMap)

	clients := make(map[string]*Client)

	for nm, cfg := range cfg.Coordinators {
		clients[nm] = NewClient(cfg)
	}

	Auth = NewAuthController(cfg, clients, vf)
	// GetTask = NewGetTaskController(cfg, chainCfg, db, vf, reg)
	// SubmitProof = NewSubmitProofController(cfg, chainCfg, db, vf, reg)
}
