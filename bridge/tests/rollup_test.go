package tests

import (
	"context"
	"math/big"
	"testing"

	"scroll-tech/common/types"
	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/bridge/l1"
	"scroll-tech/bridge/l2"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
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
	l2Relayer, err := l2.NewLayer2Relayer(context.Background(), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer l2Relayer.Stop()

	// Create L1Watcher
	l1Cfg := cfg.L1Config
	l1Watcher := l1.NewWatcher(context.Background(), l1Client, 0, l1Cfg.Confirmations, l1Cfg.L1MessengerAddress, l1Cfg.RollupContractAddress, db)

	// add some blocks to db
	var traces []*geth_types.BlockTrace
	var parentHash common.Hash
	for i := 1; i <= 10; i++ {
		header := geth_types.Header{
			Number:     big.NewInt(int64(i)),
			ParentHash: parentHash,
			Difficulty: big.NewInt(0),
			BaseFee:    big.NewInt(0),
		}
		traces = append(traces, &geth_types.BlockTrace{
			Header:       &header,
			StorageTrace: &geth_types.StorageTrace{},
		})
		parentHash = header.Hash()
	}
	err = db.InsertBlockTraces(traces)
	assert.NoError(t, err)

	// add one batch to db
	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchID, err := db.NewBatchInDBTx(dbTx,
		&types.BlockInfo{
			Number:     traces[0].Header.Number.Uint64(),
			Hash:       traces[0].Header.Hash().String(),
			ParentHash: traces[0].Header.ParentHash.String(),
		},
		&types.BlockInfo{
			Number:     traces[1].Header.Number.Uint64(),
			Hash:       traces[1].Header.Hash().String(),
			ParentHash: traces[1].Header.ParentHash.String(),
		},
		traces[0].Header.ParentHash.String(), 1, 194676) // parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = db.SetBatchIDForBlocksInDBTx(dbTx, []uint64{
		traces[0].Header.Number.Uint64(),
		traces[1].Header.Number.Uint64()}, batchID)
	assert.NoError(t, err)
	err = dbTx.Commit()
	assert.NoError(t, err)

	// process pending batch and check status
	l2Relayer.ProcessPendingBatches()

	status, err := db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupCommitting, status)
	commitTxHash, err := db.GetCommitTxHash(batchID)
	assert.NoError(t, err)
	assert.Equal(t, true, commitTxHash.Valid)
	commitTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(commitTxHash.String))
	assert.NoError(t, err)
	commitTxReceipt, err := bind.WaitMined(context.Background(), l1Client, commitTx)
	assert.NoError(t, err)
	assert.Equal(t, len(commitTxReceipt.Logs), 1)

	// fetch rollup events
	err = l1Watcher.FetchContractEvent(commitTxReceipt.BlockNumber.Uint64())
	assert.NoError(t, err)
	status, err = db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupCommitted, status)

	// add dummy proof
	tProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	tInstanceCommitments := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	err = db.UpdateProofByID(context.Background(), batchID, tProof, tInstanceCommitments, 100)
	assert.NoError(t, err)
	err = db.UpdateProvingStatus(batchID, types.ProvingTaskVerified)
	assert.NoError(t, err)

	// process committed batch and check status
	l2Relayer.ProcessCommittedBatches()

	status, err = db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupFinalizing, status)
	finalizeTxHash, err := db.GetFinalizeTxHash(batchID)
	assert.NoError(t, err)
	assert.Equal(t, true, finalizeTxHash.Valid)
	finalizeTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(finalizeTxHash.String))
	assert.NoError(t, err)
	finalizeTxReceipt, err := bind.WaitMined(context.Background(), l1Client, finalizeTx)
	assert.NoError(t, err)
	assert.Equal(t, len(finalizeTxReceipt.Logs), 1)

	// fetch rollup events
	err = l1Watcher.FetchContractEvent(finalizeTxReceipt.BlockNumber.Uint64())
	assert.NoError(t, err)
	status, err = db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupFinalized, status)
}
