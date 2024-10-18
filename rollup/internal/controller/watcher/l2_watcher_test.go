package watcher

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	cutils "scroll-tech/common/utils"

	"scroll-tech/rollup/internal/orm"
)

func setupL2Watcher(t *testing.T) (*L2WatcherClient, *gorm.DB) {
	db := setupDB(t)
	l2cfg := cfg.L2Config
	watcher := NewL2WatcherClient(context.Background(), l2Cli, l2cfg.Confirmations, l2cfg.L2MessageQueueAddress, l2cfg.WithdrawTrieRootSlot, nil, db, nil)
	return watcher, db
}

func testFetchRunningMissingBlocks(t *testing.T) {
	_, db := setupL2Watcher(t)
	defer database.CloseDB(db)

	l2BlockOrm := orm.NewL2Block(db)
	ok := cutils.TryTimes(10, func() bool {
		latestHeight, err := l2Cli.BlockNumber(context.Background())
		if err != nil {
			return false
		}
		wc := prepareWatcherClient(l2Cli, db)
		wc.TryFetchRunningMissingBlocks(latestHeight)
		fetchedHeight, err := l2BlockOrm.GetL2BlocksLatestHeight(context.Background())
		return err == nil && fetchedHeight == latestHeight
	})
	assert.True(t, ok)
}

func prepareWatcherClient(l2Cli *ethclient.Client, db *gorm.DB) *L2WatcherClient {
	confirmations := rpc.LatestBlockNumber
	return NewL2WatcherClient(context.Background(), l2Cli, confirmations, common.Address{}, common.Hash{}, nil, db, nil)
}
