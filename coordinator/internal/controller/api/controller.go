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
	// HealthCheck the health check controller
	HealthCheck *HealthCheckController
	// Auth the auth controller
	Auth *AuthController

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
		HealthCheck = NewHealthCheckController()
		GetTask = NewGetTaskController(cfg, db, vf, reg)
		SubmitProof = NewSubmitProofController(cfg, db, vf, reg)
	})
}
