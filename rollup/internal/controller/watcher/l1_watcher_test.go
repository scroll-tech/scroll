package watcher

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"

	"scroll-tech/rollup/internal/orm"
)

func setupL1Watcher(t *testing.T) (*L1WatcherClient, *gorm.DB) {
	db := setupDB(t)
	client, err := testApps.GetPoSL1Client()
	assert.NoError(t, err)
	l1Cfg := cfg.L1Config
	watcher := NewL1WatcherClient(context.Background(), client, l1Cfg.StartHeight, db, nil)
	return watcher, db
}

func testL1WatcherClientFetchBlockHeader(t *testing.T) {
	watcher, db := setupL1Watcher(t)
	defer database.CloseDB(db)
	convey.Convey("test toBlock < fromBlock", t, func() {
		var blockHeight uint64
		if watcher.ProcessedBlockHeight() <= 0 {
			blockHeight = 0
		} else {
			blockHeight = watcher.ProcessedBlockHeight() - 1
		}
		err := watcher.FetchBlockHeader(blockHeight)
		assert.NoError(t, err)
	})

	convey.Convey("test get header from client error", t, func() {
		var c *ethclient.Client
		patchGuard := gomonkey.ApplyMethodFunc(c, "HeaderByNumber", func(ctx context.Context, height *big.Int) (*types.Header, error) {
			return nil, ethereum.NotFound
		})
		defer patchGuard.Reset()

		var blockHeight uint64 = 10
		err := watcher.FetchBlockHeader(blockHeight)
		assert.Error(t, err)
	})

	var l1BlockOrm *orm.L1Block
	convey.Convey("insert l1 block error", t, func() {
		var c *ethclient.Client
		patchGuard := gomonkey.ApplyMethodFunc(c, "HeaderByNumber", func(ctx context.Context, height *big.Int) (*types.Header, error) {
			if height == nil {
				height = big.NewInt(100)
			}
			t.Log(height)
			return &types.Header{
				BaseFee: big.NewInt(100),
			}, nil
		})
		defer patchGuard.Reset()

		patchGuard.ApplyMethodFunc(l1BlockOrm, "InsertL1Blocks", func(ctx context.Context, blocks []orm.L1Block) error {
			return errors.New("insert failed")
		})

		var blockHeight uint64 = 10
		err := watcher.FetchBlockHeader(blockHeight)
		assert.Error(t, err)
	})

	convey.Convey("fetch block header success", t, func() {
		var c *ethclient.Client
		patchGuard := gomonkey.ApplyMethodFunc(c, "HeaderByNumber", func(ctx context.Context, height *big.Int) (*types.Header, error) {
			if height == nil {
				height = big.NewInt(100)
			}
			t.Log(height)
			return &types.Header{
				BaseFee: big.NewInt(100),
			}, nil
		})
		defer patchGuard.Reset()

		patchGuard.ApplyMethodFunc(l1BlockOrm, "InsertL1Blocks", func(ctx context.Context, blocks []orm.L1Block) error {
			return nil
		})

		var blockHeight uint64 = 10
		err := watcher.FetchBlockHeader(blockHeight)
		assert.NoError(t, err)
	})
}
