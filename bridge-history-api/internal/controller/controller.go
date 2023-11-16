package controller

import (
	"sync"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var (
	// HistoryCtrler is controller instance
	HistoryCtrler *HistoryController

	initControllerOnce sync.Once
)

// InitController inits Controller with database
func InitController(db *gorm.DB, redis *redis.Client) {
	initControllerOnce.Do(func() {
		HistoryCtrler = NewHistoryController(db, redis)
	})
}
