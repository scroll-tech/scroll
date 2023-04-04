package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types"

	"scroll-tech/bridge/relayer"
	"scroll-tech/bridge/watcher"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

func testCommitBatchAndFinalizeBatch(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	prepareContracts(t)

	// Create L2Relayer
	l2Cfg := cfg.L2Config
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Create L1Watcher
	l1Cfg := cfg.L1Config
	l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, 0, l1Cfg.Confirmations, l1Cfg.L1MessengerAddress, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db)

	// add some blocks to db
	var wrappedBlocks []*types.WrappedBlock
	var parentHash common.Hash
	for i := 1; i <= 10; i++ {
		header := geth_types.Header{
			Number:     big.NewInt(int64(i)),
			ParentHash: parentHash,
			Difficulty: big.NewInt(0),
			BaseFee:    big.NewInt(0),
		}
		wrappedBlocks = append(wrappedBlocks, &types.WrappedBlock{
			Header:           &header,
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		})
		parentHash = header.Hash()
	}
	assert.NoError(t, db.InsertWrappedBlocks(wrappedBlocks))

	parentBatch := &types.BlockBatch{
		Index: 0,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	batchData := types.NewBatchData(parentBatch, []*types.WrappedBlock{
		wrappedBlocks[0],
		wrappedBlocks[1],
	}, cfg.L2Config.BatchProposerConfig.PublicInputConfig)

	batchHash := batchData.Hash().String()

	// add one batch to db
	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	assert.NoError(t, db.NewBatchInDBTx(dbTx, batchData))
	var blockIDs = make([]uint64, len(batchData.Batch.Blocks))
	for i, block := range batchData.Batch.Blocks {
		blockIDs[i] = block.BlockNumber
	}
	err = db.SetBatchHashForL2BlocksInDBTx(dbTx, blockIDs, batchHash)
	assert.NoError(t, err)
	assert.NoError(t, dbTx.Commit())

	// process pending batch and check status
	l2Relayer.SendCommitTx([]*types.BatchData{batchData})

	status, err := db.GetRollupStatus(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupCommitting, status)
	commitTxHash, err := db.GetCommitTxHash(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, true, commitTxHash.Valid)
	commitTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(commitTxHash.String))
	assert.NoError(t, err)
	commitTxReceipt, err := bind.WaitMined(context.Background(), l1Client, commitTx)
	assert.NoError(t, err)
	assert.Equal(t, len(commitTxReceipt.Logs), 1)

	// fetch rollup events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	status, err = db.GetRollupStatus(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupCommitted, status)

	// add dummy proof
	tProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	tInstanceCommitments := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	err = db.UpdateProofByHash(context.Background(), batchHash, tProof, tInstanceCommitments, 100)
	assert.NoError(t, err)
	err = db.UpdateProvingStatus(batchHash, types.ProvingTaskVerified)
	assert.NoError(t, err)

	// process committed batch and check status
	l2Relayer.ProcessCommittedBatches()

	status, err = db.GetRollupStatus(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupFinalizing, status)
	finalizeTxHash, err := db.GetFinalizeTxHash(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, true, finalizeTxHash.Valid)
	finalizeTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(finalizeTxHash.String))
	assert.NoError(t, err)
	finalizeTxReceipt, err := bind.WaitMined(context.Background(), l1Client, finalizeTx)
	assert.NoError(t, err)
	assert.Equal(t, len(finalizeTxReceipt.Logs), 1)

	// fetch rollup events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	status, err = db.GetRollupStatus(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupFinalized, status)
}
