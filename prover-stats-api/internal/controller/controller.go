package controller

import (
	"sync"

	"gorm.io/gorm"
)

var (
	// ProverTask is controller instance
	ProverTask *ProverTaskController
	// Auth is controller instance
	Auth *AuthController

	initControllerOnce sync.Once
)

// InitController inits Controller with database
func InitController(db *gorm.DB) {
	initControllerOnce.Do(func() {
		ProverTask = NewProverTaskController(db)
		Auth = NewAuthController()
	})
}
