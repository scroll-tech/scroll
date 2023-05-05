package watcher

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/relayer"

	"scroll-tech/common/types"
)

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
