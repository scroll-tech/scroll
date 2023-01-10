package l1

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/ethclient/gethclient"
	"github.com/scroll-tech/go-ethereum/rpc"
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

	rawClient, err := rpc.DialContext(context.Background(), l1gethImg.Endpoint())
	assert.NoError(t, err)
	gethClient := gethclient.New(rawClient)
	ethClient := ethclient.NewClient(rawClient)

	l1Cfg := cfg.L1Config

	watcher, err := NewWatcher(context.Background(), gethClient, ethClient, l1Cfg.StartHeight, l1Cfg.Confirmations, l1Cfg.L1MessengerAddress, l1Cfg.L1MessageQueueAddress, l1Cfg.RollupContractAddress, db)
	assert.NoError(t, err)
	watcher.Start()
	defer watcher.Stop()
}
