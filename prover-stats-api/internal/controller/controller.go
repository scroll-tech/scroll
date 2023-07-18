package controller

import "gorm.io/gorm"

var (
	ProverTask *ProverTaskController
)

func InitController(db *gorm.DB) {
	ProverTask = NewProverTaskController(db)
}
