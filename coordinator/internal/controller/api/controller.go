package api

import (
	"gorm.io/gorm"
	"scroll-tech/coordinator/internal/config"
	"sync"
)

var (
	ProverTask  *ProverTaskController
	SubmitProof *SubmitProofController
	Auth        *AuthController

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
