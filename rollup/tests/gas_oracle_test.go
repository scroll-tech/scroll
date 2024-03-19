package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	"scroll-tech/common/types"
	"scroll-tech/common/types/encoding"

	"scroll-tech/rollup/internal/controller/relayer"
	"scroll-tech/rollup/internal/controller/watcher"
	"scroll-tech/rollup/internal/orm"
)

func testImportL1GasPrice(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	prepareContracts(t)

	l1Cfg := rollupApp.Config.L1Config

	// Create L1Relayer
	l1Relayer, err := relayer.NewLayer1Relayer(context.Background(), db, l1Cfg.RelayerConfig, relayer.ServiceTypeL1GasOracle, nil)
	assert.NoError(t, err)
	defer l1Relayer.StopSenders()

	// Create L1Watcher
	startHeight, err := l1Client.BlockNumber(context.Background())
	assert.NoError(t, err)
	l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, startHeight-1, 0, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db, nil)

	// fetch new blocks
	number, err := l1Client.BlockNumber(context.Background())
	assert.Greater(t, number, startHeight-1)
	assert.NoError(t, err)
	err = l1Watcher.FetchBlockHeader(number)
	assert.NoError(t, err)

	l1BlockOrm := orm.NewL1Block(db)
	// check db status
	latestBlockHeight, err := l1BlockOrm.GetLatestL1BlockHeight(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, number, latestBlockHeight)
	blocks, err := l1BlockOrm.GetL1Blocks(context.Background(), map[string]interface{}{"number": latestBlockHeight})
	assert.NoError(t, err)
	assert.Equal(t, len(blocks), 1)
	assert.Empty(t, blocks[0].OracleTxHash)
	assert.Equal(t, types.GasOracleStatus(blocks[0].GasOracleStatus), types.GasOraclePending)

	// relay gas price
	l1Relayer.ProcessGasPriceOracle()
	blocks, err = l1BlockOrm.GetL1Blocks(context.Background(), map[string]interface{}{"number": latestBlockHeight})
	assert.NoError(t, err)
	assert.Equal(t, len(blocks), 1)
	assert.NotEmpty(t, blocks[0].OracleTxHash)
	assert.Equal(t, types.GasOracleStatus(blocks[0].GasOracleStatus), types.GasOracleImporting)
}

func testImportL2GasPrice(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)
	prepareContracts(t)

	l2Cfg := rollupApp.Config.L2Config
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, &params.ChainConfig{}, false, relayer.ServiceTypeL2GasOracle, nil)
	assert.NoError(t, err)
	defer l2Relayer.StopSenders()

	// add fake chunk
	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{
			{
				Header: &gethTypes.Header{
					Number:     big.NewInt(1),
					ParentHash: common.Hash{},
					Difficulty: big.NewInt(0),
					BaseFee:    big.NewInt(0),
				},
				Transactions:   nil,
				WithdrawRoot:   common.Hash{},
				RowConsumption: &gethTypes.RowConsumption{},
			},
		},
	}
	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
	}

	batchOrm := orm.NewBatch(db)
	_, err = batchOrm.InsertBatch(context.Background(), batch)
	assert.NoError(t, err)

	// check db status
	dbBatch, err := batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, batch)
	assert.Empty(t, dbBatch.OracleTxHash)
	assert.Equal(t, types.GasOracleStatus(dbBatch.OracleStatus), types.GasOraclePending)

	// relay gas price
	l2Relayer.ProcessGasPriceOracle()
	dbBatch, err = batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, batch)
	assert.NotEmpty(t, dbBatch.OracleTxHash)
	assert.Equal(t, types.GasOracleStatus(dbBatch.OracleStatus), types.GasOracleImporting)
}
