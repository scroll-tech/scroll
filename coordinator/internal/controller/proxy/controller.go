package proxy

import (
	"github.com/prometheus/client_golang/prometheus"
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
type Clients map[string]Client

// InitController inits Controller with database
func InitController(cfg *config.ProxyConfig, reg prometheus.Registerer) {
	// normalize cfg
	cfg.ProxyManager.Normalize()

	vf, err := verifier.NewVerifier(cfg.ProxyManager.Verifier)
	if err != nil {
		panic("proof receiver new verifier failure")
	}

	log.Info("verifier created", "openVmVerifier", vf.OpenVMVkMap)

	clients := make(map[string]Client)

	for nm, upCfg := range cfg.Coordinators {
		cli, err := NewClientManager(nm, cfg.ProxyManager.Client, upCfg)
		if err != nil {
			panic("create new client fail")
		}
		clients[nm] = cli
	}

	proverManager := NewProverManager()

	Auth = NewAuthController(cfg, clients, vf, proverManager)
	GetTask = NewGetTaskController(cfg, clients, proverManager, reg)
	SubmitProof = NewSubmitProofController(cfg, clients, proverManager, reg)
}
