package watcher

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/relayer"

	"scroll-tech/common/types"
)

func testBatchProposerProposeBatch(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	p := &BatchProposer{
		batchGasThreshold:       1000,
		batchTxNumThreshold:     10,
		batchTimeSec:            300,
		commitCalldataSizeLimit: 500,
		orm:                     db,
	}
	patchGuard := gomonkey.ApplyMethodFunc(p.orm, "GetL2WrappedBlocks", func(fields map[string]interface{}, args ...string) ([]*types.WrappedBlock, error) {
		hash, _ := fields["hash"].(string)
		if hash == "blockWithLongData" {
			longData := strings.Repeat("0", 1000)
			return []*types.WrappedBlock{{
				Transactions: []*geth_types.TransactionData{{
					Data: longData,
				}},
			}}, nil
		}
		return []*types.WrappedBlock{{
			Transactions: []*geth_types.TransactionData{{
				Data: "short",
			}},
		}}, nil
	})
	defer patchGuard.Reset()
	patchGuard.ApplyPrivateMethod(p, "createBatchForBlocks", func(*BatchProposer, []*types.BlockInfo) error {
		return nil
	})

	block1 := &types.BlockInfo{Number: 1, GasUsed: 100, TxNum: 1, BlockTimestamp: uint64(time.Now().Unix()) - 200}
	block2 := &types.BlockInfo{Number: 2, GasUsed: 200, TxNum: 2, BlockTimestamp: uint64(time.Now().Unix())}
	block3 := &types.BlockInfo{Number: 3, GasUsed: 300, TxNum: 11, BlockTimestamp: uint64(time.Now().Unix())}
	block4 := &types.BlockInfo{Number: 4, GasUsed: 1001, TxNum: 3, BlockTimestamp: uint64(time.Now().Unix())}
	blockOutdated := &types.BlockInfo{Number: 1, GasUsed: 100, TxNum: 1, BlockTimestamp: uint64(time.Now().Add(-400 * time.Second).Unix())}
	blockWithLongData := &types.BlockInfo{Hash: "blockWithLongData", Number: 5, GasUsed: 500, TxNum: 1, BlockTimestamp: uint64(time.Now().Unix())}

	testCases := []struct {
		description string
		blocks      []*types.BlockInfo
		expectedRes bool
	}{
		{"Empty block list", []*types.BlockInfo{}, false},
		{"Single block exceeding gas threshold", []*types.BlockInfo{block4}, true},
		{"Single block exceeding transaction number threshold", []*types.BlockInfo{block3}, true},
		{"Multiple blocks meeting thresholds", []*types.BlockInfo{block1, block2, block3}, true},
		{"Multiple blocks not meeting thresholds", []*types.BlockInfo{block1, block2}, false},
		{"Outdated and valid block", []*types.BlockInfo{blockOutdated, block2}, true},
		{"Single block with long data", []*types.BlockInfo{blockWithLongData}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expectedRes, p.proposeBatch(tc.blocks), "Failed on: %s", tc.description)
		})
	}
}

func testBatchProposerBatchGeneration(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	ctx := context.Background()
	subCtx, cancel := context.WithCancel(ctx)

	defer func() {
		cancel()
		db.Close()
	}()

	// Insert traces into db.
	assert.NoError(t, db.InsertWrappedBlocks([]*types.WrappedBlock{wrappedBlock1}))

	l2cfg := cfg.L2Config
	wc := NewL2WatcherClient(context.Background(), l2Cli, l2cfg.Confirmations, l2cfg.L2MessengerAddress, l2cfg.L2MessageQueueAddress, l2cfg.WithdrawTrieRootSlot, db)
	loopToFetchEvent(subCtx, wc)

	batch, err := db.GetLatestBatch()
	assert.NoError(t, err)

	// Create a new batch.
	batchData := types.NewBatchData(&types.BlockBatch{
		Index:     0,
		Hash:      batch.Hash,
		StateRoot: batch.StateRoot,
	}, []*types.WrappedBlock{wrappedBlock1}, nil)

	relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	proposer := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		ProofGenerationFreq: 1,
		BatchGasThreshold:   3000000,
		BatchTxNumThreshold: 135,
		BatchTimeSec:        1,
		BatchBlocksLimit:    100,
	}, relayer, db)
	proposer.TryProposeBatch()

	infos, err := db.GetUnbatchedL2Blocks(map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", 100))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(infos))

	exist, err := db.BatchRecordExist(batchData.Hash().Hex())
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
}

func testBatchProposerGracefulRestart(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	// Insert traces into db.
	assert.NoError(t, db.InsertWrappedBlocks([]*types.WrappedBlock{wrappedBlock2}))

	// Insert block batch into db.
	batchData1 := types.NewBatchData(&types.BlockBatch{
		Index:     0,
		Hash:      common.Hash{}.String(),
		StateRoot: common.Hash{}.String(),
	}, []*types.WrappedBlock{wrappedBlock1}, nil)

	parentBatch2 := &types.BlockBatch{
		Index:     batchData1.Batch.BatchIndex,
		Hash:      batchData1.Hash().Hex(),
		StateRoot: batchData1.Batch.NewStateRoot.String(),
	}
	batchData2 := types.NewBatchData(parentBatch2, []*types.WrappedBlock{wrappedBlock2}, nil)

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	assert.NoError(t, db.NewBatchInDBTx(dbTx, batchData1))
	assert.NoError(t, db.NewBatchInDBTx(dbTx, batchData2))
	assert.NoError(t, db.SetBatchHashForL2BlocksInDBTx(dbTx, []uint64{
		batchData1.Batch.Blocks[0].BlockNumber}, batchData1.Hash().Hex()))
	assert.NoError(t, db.SetBatchHashForL2BlocksInDBTx(dbTx, []uint64{
		batchData2.Batch.Blocks[0].BlockNumber}, batchData2.Hash().Hex()))
	assert.NoError(t, dbTx.Commit())

	assert.NoError(t, db.UpdateRollupStatus(context.Background(), batchData1.Hash().Hex(), types.RollupFinalized))

	batchHashes, err := db.GetPendingBatches(math.MaxInt32)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(batchHashes))
	assert.Equal(t, batchData2.Hash().Hex(), batchHashes[0])
	// test p.recoverBatchDataBuffer().
	_ = NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		ProofGenerationFreq: 1,
		BatchGasThreshold:   3000000,
		BatchTxNumThreshold: 135,
		BatchTimeSec:        1,
		BatchBlocksLimit:    100,
	}, relayer, db)

	batchHashes, err = db.GetPendingBatches(math.MaxInt32)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(batchHashes))

	exist, err := db.BatchRecordExist(batchData2.Hash().Hex())
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
}
