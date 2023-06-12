package watcher

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum/common"
	gethTtypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/controller/relayer"
	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
	bridgeUtils "scroll-tech/bridge/internal/utils"
)

func testBatchProposerProposeBatch(t *testing.T) {
	db := setupDB(t)
	defer bridgeUtils.CloseDB(db)

	p := &BatchProposer{
		batchGasThreshold:       1000,
		batchTxNumThreshold:     10,
		batchTimeSec:            300,
		commitCalldataSizeLimit: 500,
	}

	var blockTrace *orm.BlockTrace
	patchGuard := gomonkey.ApplyMethodFunc(blockTrace, "GetL2WrappedBlocks", func(fields map[string]interface{}) ([]*bridgeTypes.WrappedBlock, error) {
		hash, _ := fields["hash"].(string)
		if hash == "blockWithLongData" {
			longData := strings.Repeat("0", 1000)
			return []*bridgeTypes.WrappedBlock{{
				Transactions: []*gethTtypes.TransactionData{{
					Data: longData,
				}},
			}}, nil
		}
		return []*bridgeTypes.WrappedBlock{{
			Transactions: []*gethTtypes.TransactionData{{
				Data: "short",
			}},
		}}, nil
	})
	defer patchGuard.Reset()
	patchGuard.ApplyPrivateMethod(p, "createBatchForBlocks", func(*BatchProposer, []*types.BlockInfo) error {
		return nil
	})

	block1 := orm.BlockTrace{Number: 1, GasUsed: 100, TxNum: 1, BlockTimestamp: uint64(time.Now().Unix()) - 200}
	block2 := orm.BlockTrace{Number: 2, GasUsed: 200, TxNum: 2, BlockTimestamp: uint64(time.Now().Unix())}
	block3 := orm.BlockTrace{Number: 3, GasUsed: 300, TxNum: 11, BlockTimestamp: uint64(time.Now().Unix())}
	block4 := orm.BlockTrace{Number: 4, GasUsed: 1001, TxNum: 3, BlockTimestamp: uint64(time.Now().Unix())}
	blockOutdated := orm.BlockTrace{Number: 1, GasUsed: 100, TxNum: 1, BlockTimestamp: uint64(time.Now().Add(-400 * time.Second).Unix())}
	blockWithLongData := orm.BlockTrace{Hash: "blockWithLongData", Number: 5, GasUsed: 500, TxNum: 1, BlockTimestamp: uint64(time.Now().Unix())}

	testCases := []struct {
		description string
		blocks      []orm.BlockTrace
		expectedRes bool
	}{
		{"Empty block list", []orm.BlockTrace{}, false},
		{"Single block exceeding gas threshold", []orm.BlockTrace{block4}, true},
		{"Single block exceeding transaction number threshold", []orm.BlockTrace{block3}, true},
		{"Multiple blocks meeting thresholds", []orm.BlockTrace{block1, block2, block3}, true},
		{"Multiple blocks not meeting thresholds", []orm.BlockTrace{block1, block2}, false},
		{"Outdated and valid block", []orm.BlockTrace{blockOutdated, block2}, true},
		{"Single block with long data", []orm.BlockTrace{blockWithLongData}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expectedRes, p.proposeBatch(tc.blocks), "Failed on: %s", tc.description)
		})
	}
}

func testBatchProposerBatchGeneration(t *testing.T) {
	db := setupDB(t)
	subCtx, cancel := context.WithCancel(context.Background())
	defer func() {
		bridgeUtils.CloseDB(db)
		cancel()
	}()
	blockTraceOrm := orm.NewBlockTrace(db)
	// Insert traces into db.
	assert.NoError(t, blockTraceOrm.InsertWrappedBlocks([]*bridgeTypes.WrappedBlock{wrappedBlock1}))

	l2cfg := cfg.L2Config
	wc := NewL2WatcherClient(context.Background(), l2Cli, l2cfg.Confirmations, l2cfg.L2MessengerAddress, l2cfg.L2MessageQueueAddress, l2cfg.WithdrawTrieRootSlot, db)
	loopToFetchEvent(subCtx, wc)

	blockBatchOrm := orm.NewBlockBatch(db)
	batch, err := blockBatchOrm.GetLatestBatch()
	assert.NoError(t, err)

	// Create a new batch.
	batchData := bridgeTypes.NewBatchData(&bridgeTypes.BatchInfo{
		Index:     0,
		Hash:      batch.Hash,
		StateRoot: batch.StateRoot,
	}, []*bridgeTypes.WrappedBlock{wrappedBlock1}, nil)

	relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	proposer := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		ProofGenerationFreq:     1,
		BatchGasThreshold:       3000000,
		BatchTxNumThreshold:     135,
		BatchTimeSec:            1,
		BatchBlocksLimit:        100,
		CommitTxBatchCountLimit: 30,
	}, relayer, db)
	proposer.TryProposeBatch()

	infos, err := blockTraceOrm.GetUnbatchedL2Blocks(map[string]interface{}{}, []string{"number ASC"}, 100)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(infos))

	batches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": batchData.Hash().Hex()}, nil, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(batches))
}

func testBatchProposerGracefulRestart(t *testing.T) {
	db := setupDB(t)
	defer bridgeUtils.CloseDB(db)

	relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	blockTraceOrm := orm.NewBlockTrace(db)
	// Insert traces into db.
	assert.NoError(t, blockTraceOrm.InsertWrappedBlocks([]*bridgeTypes.WrappedBlock{wrappedBlock2}))

	// Insert block batch into db.
	parentBatch1 := &bridgeTypes.BatchInfo{
		Index:     0,
		Hash:      common.Hash{}.String(),
		StateRoot: common.Hash{}.String(),
	}
	batchData1 := bridgeTypes.NewBatchData(parentBatch1, []*bridgeTypes.WrappedBlock{wrappedBlock1}, nil)

	parentBatch2 := &bridgeTypes.BatchInfo{
		Index:     batchData1.Batch.BatchIndex,
		Hash:      batchData1.Hash().Hex(),
		StateRoot: batchData1.Batch.NewStateRoot.String(),
	}
	batchData2 := bridgeTypes.NewBatchData(parentBatch2, []*bridgeTypes.WrappedBlock{wrappedBlock2}, nil)

	blockBatchOrm := orm.NewBlockBatch(db)
	err = db.Transaction(func(tx *gorm.DB) error {
		_, dbTxErr := blockBatchOrm.InsertBlockBatchByBatchData(tx, batchData1)
		if dbTxErr != nil {
			return dbTxErr
		}
		_, dbTxErr = blockBatchOrm.InsertBlockBatchByBatchData(tx, batchData2)
		if dbTxErr != nil {
			return dbTxErr
		}
		numbers1 := []uint64{batchData1.Batch.Blocks[0].BlockNumber}
		hash1 := batchData1.Hash().Hex()
		dbTxErr = blockTraceOrm.UpdateChunkHashForL2Blocks(tx, numbers1, hash1)
		if dbTxErr != nil {
			return dbTxErr
		}
		numbers2 := []uint64{batchData2.Batch.Blocks[0].BlockNumber}
		hash2 := batchData2.Hash().Hex()
		dbTxErr = blockTraceOrm.UpdateChunkHashForL2Blocks(tx, numbers2, hash2)
		if dbTxErr != nil {
			return dbTxErr
		}
		return nil
	})
	assert.NoError(t, err)
	err = blockBatchOrm.UpdateRollupStatus(context.Background(), batchData1.Hash().Hex(), types.RollupFinalized)
	assert.NoError(t, err)
	batchHashes, err := blockBatchOrm.GetBlockBatchesHashByRollupStatus(types.RollupPending, math.MaxInt32)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(batchHashes))
	assert.Equal(t, batchData2.Hash().Hex(), batchHashes[0])
	// test p.recoverBatchDataBuffer().
	_ = NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		ProofGenerationFreq:     1,
		BatchGasThreshold:       3000000,
		BatchTxNumThreshold:     135,
		BatchTimeSec:            1,
		BatchBlocksLimit:        100,
		CommitTxBatchCountLimit: 30,
	}, relayer, db)

	batchHashes, err = blockBatchOrm.GetBlockBatchesHashByRollupStatus(types.RollupPending, math.MaxInt32)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(batchHashes))

	batches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": batchData2.Hash().Hex()}, nil, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(batches))
}
