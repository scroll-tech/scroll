package api

import (
	"sync"

	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/config"
)

var (
	// ProverTask the prover task controller
	ProverTask *ProverTaskController
	// SubmitProof the submit proof controller
	SubmitProof *SubmitProofController
	// Auth the auth controller
	Auth *AuthController

	initControllerOnce sync.Once
)

// InitController inits Controller with database
func InitController(cfg *config.Config, db *gorm.DB) {
	initControllerOnce.Do(func() {
		Auth = NewAuthController()
		ProverTask = NewProverTaskController(cfg, db)
		SubmitProof = NewSubmitProofController(cfg, db)
	})
}
