package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	"scroll-tech/common/types"

	"scroll-tech/bridge/internal/controller/relayer"
	"scroll-tech/bridge/internal/controller/watcher"
	"scroll-tech/bridge/internal/orm"
)

func testImportL1GasPrice(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	prepareContracts(t)

	l1Cfg := bridgeApp.Config.L1Config

	// Create L1Relayer
	l1Relayer, err := relayer.NewLayer1Relayer(context.Background(), db, l1Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Create L1Watcher
	startHeight, err := l1Client.BlockNumber(context.Background())
	assert.NoError(t, err)
	l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, startHeight-1, 0, l1Cfg.L1MessengerAddress, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db)

	// fetch new blocks
	number, err := l1Client.BlockNumber(context.Background())
	assert.Greater(t, number, startHeight-1)
	assert.NoError(t, err)
	err = l1Watcher.FetchBlockHeader(number)
	assert.NoError(t, err)

	l1BlockOrm := orm.NewL1Block(db)
	// check db status
	latestBlockHeight, err := l1BlockOrm.GetLatestL1BlockHeight()
	assert.NoError(t, err)
	assert.Equal(t, number, latestBlockHeight)
	blocks, err := l1BlockOrm.GetL1Blocks(map[string]interface{}{"number": latestBlockHeight})
	assert.NoError(t, err)
	assert.Equal(t, len(blocks), 1)
	assert.Empty(t, blocks[0].OracleTxHash)
	assert.Equal(t, types.GasOracleStatus(blocks[0].GasOracleStatus), types.GasOraclePending)

	// relay gas price
	l1Relayer.ProcessGasPriceOracle()
	blocks, err = l1BlockOrm.GetL1Blocks(map[string]interface{}{"number": latestBlockHeight})
	assert.NoError(t, err)
	assert.Equal(t, len(blocks), 1)
	assert.NotEmpty(t, blocks[0].OracleTxHash)
	assert.Equal(t, types.GasOracleStatus(blocks[0].GasOracleStatus), types.GasOracleImporting)
}

func testImportL2GasPrice(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)
	prepareContracts(t)

	l2Cfg := bridgeApp.Config.L2Config
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, false)
	assert.NoError(t, err)

	// add fake chunk
	chunk := &types.Chunk{
		Blocks: []*types.WrappedBlock{
			{
				Header: &gethTypes.Header{
					Number:     big.NewInt(1),
					ParentHash: common.Hash{},
					Difficulty: big.NewInt(0),
					BaseFee:    big.NewInt(0),
				},
				Transactions:     nil,
				WithdrawTrieRoot: common.Hash{},
			},
		},
	}
	chunkHash, err := chunk.Hash(0)
	assert.NoError(t, err)

	batchOrm := orm.NewBatch(db)
	_, err = batchOrm.InsertBatch(context.Background(), 0, 0, chunkHash.Hex(), chunkHash.Hex(), []*types.Chunk{chunk})
	assert.NoError(t, err)

	// check db status
	batch, err := batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, batch.OracleTxHash)
	assert.Equal(t, types.GasOracleStatus(batch.OracleStatus), types.GasOraclePending)

	// relay gas price
	l2Relayer.ProcessGasPriceOracle()
	batch, err = batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, batch.OracleTxHash)
	assert.Equal(t, types.GasOracleStatus(batch.OracleStatus), types.GasOracleImporting)
}
