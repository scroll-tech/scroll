package watcher

import (
	"context"
	"os"
	"testing"

	"gorm.io/gorm"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	cutils "scroll-tech/common/utils"

	"scroll-tech/rollup/internal/orm"
)

func setupL2Watcher(t *testing.T) (*L2WatcherClient, *gorm.DB) {
	db := setupDB(t)
	l2cfg := cfg.L2Config
	watcher := NewL2WatcherClient(context.Background(), l2Rpc, l2cfg.Confirmations, l2cfg.L2MessageQueueAddress, l2cfg.WithdrawTrieRootSlot, nil, db, false, nil)
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
		wc := prepareWatcherClient(l2Rpc, db)
		wc.TryFetchRunningMissingBlocks(latestHeight)
		fetchedHeight, err := l2BlockOrm.GetL2BlocksLatestHeight(context.Background())
		return err == nil && fetchedHeight == latestHeight
	})
	assert.True(t, ok)
}

func prepareWatcherClient(l2Cli *rpc.Client, db *gorm.DB) *L2WatcherClient {
	confirmations := rpc.LatestBlockNumber
	return NewL2WatcherClient(context.Background(), l2Cli, confirmations, common.Address{}, common.Hash{}, nil, db, false, nil)
}

// New test for raw RPC GetBlockByHash from an endpoint URL in env.
func TestRawRPCGetBlockByHash(t *testing.T) {
	url := os.Getenv("RPC_ENDPOINT_URL")
	if url == "" {
		t.Log("warn: RPC_ENDPOINT_URL not set, skipping raw RPC test")
		t.Skip("missing RPC_ENDPOINT_URL")
	}

	ctx := context.Background()
	cli, err := rpc.DialContext(ctx, url)
	if err != nil {
		t.Fatalf("failed to dial RPC endpoint %s: %v", url, err)
	}
	defer cli.Close()

	var txs []*types.Transaction
	blkHash := common.HexToHash("0xc80cf12883341827d71c08f734ba9a9d6da7e59eb16921d26e6706887e552c74")
	err = cli.CallContext(ctx, &txs, "scroll_getL1MessagesInBlock", blkHash, "synced")
	if err != nil {
		t.Logf("scroll_getL1MessagesInBlock failed: err=%v", err)
		t.Fail()
	}
	t.Log(txs, txs[0].Hash())
}
