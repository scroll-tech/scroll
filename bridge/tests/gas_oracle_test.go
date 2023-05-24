package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	"scroll-tech/bridge/internal/controller/relayer"
	"scroll-tech/bridge/internal/controller/watcher"
	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
	"scroll-tech/bridge/internal/utils"
)

func testImportL1GasPrice(t *testing.T) {
	db := setupDB(t)
	defer utils.CloseDB(db)

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
	assert.Equal(t, int64(number), latestBlockHeight)
	blocks, err := l1BlockOrm.GetL1BlockInfos(map[string]interface{}{"number": latestBlockHeight})
	assert.NoError(t, err)
	assert.Equal(t, len(blocks), 1)
	assert.Equal(t, types.GasOracleStatus(blocks[0].GasOracleStatus), types.GasOraclePending)

	// relay gas price
	l1Relayer.ProcessGasPriceOracle()
	blocks, err = l1BlockOrm.GetL1BlockInfos(map[string]interface{}{"number": latestBlockHeight})
	assert.NoError(t, err)
	assert.Equal(t, len(blocks), 1)
	assert.Equal(t, types.GasOracleStatus(blocks[0].GasOracleStatus), types.GasOracleImporting)
}

func testImportL2GasPrice(t *testing.T) {
	db := setupDB(t)
	defer utils.CloseDB(db)
	prepareContracts(t)

	l2Cfg := bridgeApp.Config.L2Config
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// add fake blocks
	traces := []*bridgeTypes.WrappedBlock{
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
	}

	blockTraceOrm := orm.NewBlockTrace(db)
	assert.NoError(t, blockTraceOrm.InsertWrappedBlocks(traces))

	parentBatch := &bridgeTypes.WrappedBlockBatch{
		Index: 0,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	batchData := bridgeTypes.NewBatchData(parentBatch, []*bridgeTypes.WrappedBlock{traces[0]}, l2Cfg.BatchProposerConfig.PublicInputConfig)
	blockBatchOrm := orm.NewBlockBatch(db)
	err = db.Transaction(func(tx *gorm.DB) error {
		_, dbTxErr := blockBatchOrm.InsertBlockBatchByBatchData(tx, batchData)
		if dbTxErr != nil {
			return dbTxErr
		}
		return nil
	})
	assert.NoError(t, err)

	// check db status
	batch, err := blockBatchOrm.GetLatestBatch()
	assert.NoError(t, err)
	assert.Equal(t, types.GasOracleStatus(batch.OracleStatus), types.GasOraclePending)

	// relay gas price
	l2Relayer.ProcessGasPriceOracle()
	batch, err = blockBatchOrm.GetLatestBatch()
	assert.NoError(t, err)
	assert.Equal(t, types.GasOracleStatus(batch.OracleStatus), types.GasOracleImporting)
}
