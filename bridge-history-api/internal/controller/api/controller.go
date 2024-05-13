package api

import (
	"sync"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var (
	// TxsByAddressCtl the TxsByAddressController instance
	TxsByAddressCtl *TxsByAddressController

	// TxsByHashesCtl the TxsByHashesController instance
	TxsByHashesCtl *TxsByHashesController

	// L2UnClaimedWithdrawalsByAddressCtl the L2UnclaimedWithdrawalsByAddressController instance
	L2UnClaimedWithdrawalsByAddressCtl *L2UnclaimedWithdrawalsByAddressController

	// L2WithdrawalsByAddressCtl the L2WithdrawalsByAddressController instance
	L2WithdrawalsByAddressCtl *L2WithdrawalsByAddressController

	initControllerOnce sync.Once
)

// InitController inits Controller with database
func InitController(db *gorm.DB, redis *redis.Client) {
	initControllerOnce.Do(func() {
		TxsByAddressCtl = NewTxsByAddressController(db, redis)
		TxsByHashesCtl = NewTxsByHashesController(db, redis)
		L2UnClaimedWithdrawalsByAddressCtl = NewL2UnclaimedWithdrawalsByAddressController(db, redis)
		L2WithdrawalsByAddressCtl = NewL2WithdrawalsByAddressController(db, redis)
	})
}
