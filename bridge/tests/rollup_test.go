package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

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

	// add some blocks to db
	var wrappedBlocks []*bridgeTypes.WrappedBlock
	for i := 0; i < 10; i++ {
		header := gethTypes.Header{
			Number:     big.NewInt(int64(i)),
			ParentHash: common.Hash{},
			Difficulty: big.NewInt(0),
			BaseFee:    big.NewInt(0),
		}
		wrappedBlocks = append(wrappedBlocks, &bridgeTypes.WrappedBlock{
			Header:           &header,
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		})
	}

	l2BlockOrm := orm.NewL2Block(db)
	err = l2BlockOrm.InsertL2Blocks(wrappedBlocks)
	assert.NoError(t, err)

	chunkOrm := orm.NewChunk(db)
	chunk := &bridgeTypes.Chunk{Blocks: wrappedBlocks}
	chunkHash, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)

	batchOrm := orm.NewBatch(db)
	batchHash, err := batchOrm.InsertBatch(context.Background(), 0, 0, chunkHash, chunkHash, []*bridgeTypes.Chunk{chunk})
	assert.NoError(t, err)

	l2Relayer.ProcessPendingBatches()

	batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{"hash": batchHash}, nil, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(batches))
	assert.NotEmpty(t, batches[0].CommitTxHash)
	assert.Equal(t, types.RollupStatus(batches[0].RollupStatus), types.RollupCommitting)

	assert.NoError(t, err)
	commitTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(batches[0].CommitTxHash))
	assert.NoError(t, err)
	commitTxReceipt, err := bind.WaitMined(context.Background(), l1Client, commitTx)
	assert.NoError(t, err)
	assert.Equal(t, len(commitTxReceipt.Logs), 1)

	// fetch rollup events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batchHash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupCommitted, statuses[0])

	// add dummy proof
	proof := &message.AggProof{
		Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		FinalPair: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
	}
	err = batchOrm.UpdateProofByHash(context.Background(), batchHash, proof, 100)
	assert.NoError(t, err)
	err = batchOrm.UpdateProvingStatus(context.Background(), batchHash, types.ProvingTaskVerified)
	assert.NoError(t, err)

	// process committed batch and check status
	l2Relayer.ProcessCommittedBatches()

	statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batchHash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupFinalizing, statuses[0])

	batches, err = batchOrm.GetBatches(context.Background(), map[string]interface{}{"hash": batchHash}, nil, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(batches))
	assert.NotEmpty(t, batches[0].FinalizeTxHash)

	finalizeTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(batches[0].FinalizeTxHash))
	assert.NoError(t, err)
	finalizeTxReceipt, err := bind.WaitMined(context.Background(), l1Client, finalizeTx)
	assert.NoError(t, err)
	assert.Equal(t, len(finalizeTxReceipt.Logs), 1)

	// fetch rollup events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batchHash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupFinalized, statuses[0])
}
