package api

import (
	"github.com/prometheus/client_golang/prometheus"
	"sync"

	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/config"
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
		Auth = NewAuthController(db)
		HealthCheck = NewHealthCheckController()
		GetTask = NewGetTaskController(cfg, db)
		SubmitProof = NewSubmitProofController(cfg, db, reg)
	})
}
