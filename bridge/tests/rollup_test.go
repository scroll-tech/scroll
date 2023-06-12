package tests

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/bridge/internal/controller/relayer"
	"scroll-tech/bridge/internal/controller/watcher"
	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
	"scroll-tech/bridge/internal/utils"
)

func testCommitBatchAndFinalizeBatch(t *testing.T) {
	db := setupDB(t)
	defer utils.CloseDB(db)

	prepareContracts(t)

	// Create L2Relayer
	l2Cfg := bridgeApp.Config.L2Config
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Create L1Watcher
	l1Cfg := bridgeApp.Config.L1Config
	l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, 0, l1Cfg.Confirmations, l1Cfg.L1MessengerAddress, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db)

	blockTraceOrm := orm.NewBlockTrace(db)

	// add some blocks to db
	var wrappedBlocks []*bridgeTypes.WrappedBlock
	var parentHash common.Hash
	for i := 1; i <= 10; i++ {
		header := gethTypes.Header{
			Number:     big.NewInt(int64(i)),
			ParentHash: parentHash,
			Difficulty: big.NewInt(0),
			BaseFee:    big.NewInt(0),
		}
		wrappedBlocks = append(wrappedBlocks, &bridgeTypes.WrappedBlock{
			Header:           &header,
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		})
		parentHash = header.Hash()
	}
	assert.NoError(t, blockTraceOrm.InsertWrappedBlocks(wrappedBlocks))

	parentBatch := &bridgeTypes.BatchInfo{
		Index: 0,
		Hash:  "0x0000000000000000000000000000000000000000",
	}

	tmpWrapBlocks := []*bridgeTypes.WrappedBlock{
		wrappedBlocks[0],
		wrappedBlocks[1],
	}
	batchData := bridgeTypes.NewBatchData(parentBatch, tmpWrapBlocks, l2Cfg.BatchProposerConfig.PublicInputConfig)

	batchHash := batchData.Hash().String()

	blockBatchOrm := orm.NewBlockBatch(db)
	err = db.Transaction(func(tx *gorm.DB) error {
		rowsAffected, dbTxErr := blockBatchOrm.InsertBlockBatchByBatchData(tx, batchData)
		if dbTxErr != nil {
			return dbTxErr
		}
		if rowsAffected != 1 {
			dbTxErr = errors.New("the InsertBlockBatchByBatchData affected row is not 1")
			return dbTxErr
		}
		var blockIDs = make([]uint64, len(batchData.Batch.Blocks))
		for i, block := range batchData.Batch.Blocks {
			blockIDs[i] = block.BlockNumber
		}
		dbTxErr = blockTraceOrm.UpdateChunkHashForL2Blocks(tx, blockIDs, batchHash)
		if dbTxErr != nil {
			return dbTxErr
		}
		return nil
	})
	assert.NoError(t, err)

	// process pending batch and check status
	assert.NoError(t, l2Relayer.SendCommitTx([]*bridgeTypes.BatchData{batchData}))

	blockBatches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": batchHash}, nil, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blockBatches))
	assert.NotEmpty(t, true, blockBatches[0].CommitTxHash)
	assert.NotEmpty(t, true, blockBatches[0].RollupStatus)
	assert.Equal(t, types.RollupStatus(blockBatches[0].RollupStatus), types.RollupCommitting)

	commitTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(blockBatches[0].CommitTxHash))
	assert.NoError(t, err)
	commitTxReceipt, err := bind.WaitMined(context.Background(), l1Client, commitTx)
	assert.NoError(t, err)
	assert.Equal(t, len(commitTxReceipt.Logs), 1)

	// fetch rollup events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	statuses, err := blockBatchOrm.GetRollupStatusByHashList([]string{batchHash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupCommitted, statuses[0])

	// add dummy proof
	proof := &message.AggProof{
		Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		FinalPair: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
	}
	err = blockBatchOrm.UpdateProofByHash(context.Background(), batchHash, proof, 100)
	assert.NoError(t, err)
	err = blockBatchOrm.UpdateProvingStatus(batchHash, types.ProvingTaskVerified)
	assert.NoError(t, err)

	// process committed batch and check status
	l2Relayer.ProcessCommittedBatches()

	statuses, err = blockBatchOrm.GetRollupStatusByHashList([]string{batchHash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupFinalizing, statuses[0])

	blockBatches, err = blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": batchHash}, nil, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blockBatches))
	assert.NotEmpty(t, blockBatches[0].FinalizeTxHash)

	finalizeTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(blockBatches[0].FinalizeTxHash))
	assert.NoError(t, err)
	finalizeTxReceipt, err := bind.WaitMined(context.Background(), l1Client, finalizeTx)
	assert.NoError(t, err)
	assert.Equal(t, len(finalizeTxReceipt.Logs), 1)

	// fetch rollup events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	statuses, err = blockBatchOrm.GetRollupStatusByHashList([]string{batchHash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupFinalized, statuses[0])
}
