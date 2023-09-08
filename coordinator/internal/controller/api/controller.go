package api

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"

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
	// HealthCheck the health check controller
	HealthCheck *HealthCheckController
	// Ready the ready controller
	Ready *ReadyController

	initControllerOnce sync.Once
)

// InitController inits Controller with database
func InitController(cfg *config.Config, db *gorm.DB, reg prometheus.Registerer) {
	initControllerOnce.Do(func() {
		vf, err := verifier.NewVerifier(cfg.ProverManager.Verifier)
		if err != nil {
			panic("proof receiver new verifier failure")
		}

		Auth = NewAuthController(db)
		HealthCheck = NewHealthCheckController(db)
		GetTask = NewGetTaskController(cfg, db, vf, reg)
		SubmitProof = NewSubmitProofController(cfg, db, vf, reg)
		Ready = NewReadyController()
	})
}
