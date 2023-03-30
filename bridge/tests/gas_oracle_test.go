package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types"

	"scroll-tech/bridge/relayer"
	"scroll-tech/bridge/watcher"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

func testImportL1GasPrice(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	prepareContracts(t)

	l1Cfg := cfg.L1Config

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

	// check db status
	latestBlockHeight, err := db.GetLatestL1BlockHeight()
	assert.NoError(t, err)
	assert.Equal(t, number, latestBlockHeight)
	blocks, err := db.GetL1BlockInfos(map[string]interface{}{
		"number": latestBlockHeight,
	})
	assert.NoError(t, err)
	assert.Equal(t, len(blocks), 1)
	assert.Equal(t, blocks[0].GasOracleStatus, types.GasOraclePending)
	assert.Equal(t, blocks[0].OracleTxHash.Valid, false)

	// relay gas price
	l1Relayer.ProcessGasPriceOracle()
	blocks, err = db.GetL1BlockInfos(map[string]interface{}{
		"number": latestBlockHeight,
	})
	assert.NoError(t, err)
	assert.Equal(t, len(blocks), 1)
	assert.Equal(t, blocks[0].GasOracleStatus, types.GasOracleImporting)
	assert.Equal(t, blocks[0].OracleTxHash.Valid, true)
}

func testImportL2GasPrice(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	prepareContracts(t)

	l2Cfg := cfg.L2Config

	// Create L2Relayer
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// add fake blocks
	traces := []*types.WrappedBlock{
		{
			Header: &geth_types.Header{
				Number:     big.NewInt(1),
				ParentHash: common.Hash{},
				Difficulty: big.NewInt(0),
				BaseFee:    big.NewInt(0),
			},
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		},
	}
	assert.NoError(t, db.InsertWrappedBlocks(traces))

	parentBatch := &types.BlockBatch{
		Index: 0,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	batchData := types.NewBatchData(parentBatch, []*types.WrappedBlock{
		traces[0],
	}, cfg.L2Config.BatchProposerConfig.PublicInputConfig)

	// add fake batch
	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	assert.NoError(t, db.NewBatchInDBTx(dbTx, batchData))
	assert.NoError(t, dbTx.Commit())

	// check db status
	batch, err := db.GetLatestBatch()
	assert.NoError(t, err)
	assert.Equal(t, batch.OracleStatus, types.GasOraclePending)
	assert.Equal(t, batch.OracleTxHash.Valid, false)

	// relay gas price
	l2Relayer.ProcessGasPriceOracle()
	batch, err = db.GetLatestBatch()
	assert.NoError(t, err)
	assert.Equal(t, batch.OracleStatus, types.GasOracleImporting)
	assert.Equal(t, batch.OracleTxHash.Valid, true)
}
