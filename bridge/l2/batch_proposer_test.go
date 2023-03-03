package l2

import (
	"context"
	"fmt"
	"math"
	"testing"

	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/bridge/config"

	"scroll-tech/common/types"
)

func testBatchProposerProposeBatch(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	// Insert traces into db.
	_, err = db.InsertL2BlockTraces([]*geth_types.BlockTrace{blockTrace1})
	assert.NoError(t, err)

	l2cfg := cfg.L2Config
	wc := NewL2WatcherClient(context.Background(), l2Cli, l2cfg.Confirmations, l2cfg.L2MessengerAddress, l2cfg.L2MessageQueueAddress, db)
	wc.Start()
	defer wc.Stop()

	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	proposer := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		ProofGenerationFreq: 1,
		BatchGasThreshold:   3000000,
		BatchTxNumThreshold: 135,
		BatchTimeSec:        1,
		BatchBlocksLimit:    100,
	}, relayer, db)
	proposer.tryProposeBatch()

	infos, err := db.GetUnbatchedL2Blocks(map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", 100))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(infos))

	exist, err := db.BatchRecordExist(batchData1.Hash().Hex())
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
}

func testBatchProposerGracefulRestart(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	// Insert traces into db.
	_, err = db.InsertL2BlockTraces([]*geth_types.BlockTrace{blockTrace2})
	assert.NoError(t, err)

	// Insert block batch into db.
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
