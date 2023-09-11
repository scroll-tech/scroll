package controller

import (
	"sync"

	"gorm.io/gorm"
)

var (
	// HistoryCtrler is controller instance
	HistoryCtrler *HistoryController
	// BatchCtrler is controller instance
	BatchCtrler *BatchController
	// HealthCheck the health check controller
	HealthCheck *HealthCheckController
	// Ready the ready controller
	Ready *ReadyController

	initControllerOnce sync.Once
)

// InitController inits Controller with database
func InitController(db *gorm.DB) {
	initControllerOnce.Do(func() {
		HistoryCtrler = NewHistoryController(db)
		BatchCtrler = NewBatchController(db)
		HealthCheck = NewHealthCheckController(db)
		Ready = NewReadyController()
	})
}
