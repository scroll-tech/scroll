package controller

import (
	"sync"

	"gorm.io/gorm"
)

var (
	ProverTask *ProverTaskController
	Auth       *AuthController

	initControllerOnce sync.Once
)

func InitController(db *gorm.DB) {
	initControllerOnce.Do(func() {
		ProverTask = NewProverTaskController(db)
		Auth = NewAuthController()
	})
}
