package proxy

import (
	"github.com/prometheus/client_golang/prometheus"

	"scroll-tech/coordinator/internal/config"
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

	clients := make(map[string]Client)

	for nm, upCfg := range cfg.Coordinators {
		cli, err := NewClientManager(nm, cfg.ProxyManager.Client, upCfg)
		if err != nil {
			panic("create new client fail")
		}
		clients[nm] = cli
	}

	proverManager := NewProverManager()
	priorityManager := NewPriorityUpstreamManager()

	Auth = NewAuthController(cfg, clients, proverManager)
	GetTask = NewGetTaskController(cfg, clients, proverManager, priorityManager, reg)
	SubmitProof = NewSubmitProofController(cfg, clients, proverManager, reg)
}
