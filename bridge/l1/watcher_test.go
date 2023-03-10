package l1

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

func testStartWatcher(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	client, err := ethclient.Dial(base.L1GethEndpoint())
	assert.NoError(t, err)

	l1Cfg := cfg.L1Config

	watcher := NewWatcher(context.Background(), client, l1Cfg.StartHeight, l1Cfg.Confirmations, l1Cfg.L1MessengerAddress, l1Cfg.L1MessageQueueAddress, l1Cfg.RelayerConfig.RollupContractAddress, db)
	watcher.Start()
	defer watcher.Stop()
}
