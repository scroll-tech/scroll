package orm

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/database/migrate"
)

var (
	base *docker.App

	db                    *gorm.DB
	l2BlockOrm            *L2Block
	chunkOrm              *Chunk
	batchOrm              *Batch
	pendingTransactionOrm *PendingTransaction

	wrappedBlock1 *types.WrappedBlock
	wrappedBlock2 *types.WrappedBlock
	chunk1        *types.Chunk
	chunk2        *types.Chunk
	chunkHash1    common.Hash
	chunkHash2    common.Hash
)

func TestMain(m *testing.M) {
	t := &testing.T{}
	setupEnv(t)
	defer tearDownEnv(t)
	m.Run()
}

func setupEnv(t *testing.T) {
	base = docker.NewDockerApp()
	base.RunDBImage(t)
	var err error
	db, err = database.InitDB(
		&database.Config{
			DSN:        base.DBConfig.DSN,
			DriverName: base.DBConfig.DriverName,
			MaxOpenNum: base.DBConfig.MaxOpenNum,
			MaxIdleNum: base.DBConfig.MaxIdleNum,
		},
	)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	batchOrm = NewBatch(db)
	chunkOrm = NewChunk(db)
	l2BlockOrm = NewL2Block(db)
	pendingTransactionOrm = NewPendingTransaction(db)

	templateBlockTrace, err := os.ReadFile("../../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	wrappedBlock1 = &types.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace, wrappedBlock1)
	assert.NoError(t, err)

	templateBlockTrace, err = os.ReadFile("../../../common/testdata/blockTrace_03.json")
	assert.NoError(t, err)
	wrappedBlock2 = &types.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace, wrappedBlock2)
	assert.NoError(t, err)

	chunk1 = &types.Chunk{Blocks: []*types.WrappedBlock{wrappedBlock1}}
	chunkHash1, err = chunk1.Hash(0)
	assert.NoError(t, err)

	chunk2 = &types.Chunk{Blocks: []*types.WrappedBlock{wrappedBlock2}}
	chunkHash2, err = chunk2.Hash(chunk1.NumL1Messages(0))
	assert.NoError(t, err)
}

func tearDownEnv(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	sqlDB.Close()
	base.Free()
}

func TestL1BlockOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	l1BlockOrm := NewL1Block(db)

	// mock blocks
	block1 := L1Block{Number: 1, Hash: "hash1"}
	block2 := L1Block{Number: 2, Hash: "hash2"}
	block3 := L1Block{Number: 3, Hash: "hash3"}
	block2AfterReorg := L1Block{Number: 2, Hash: "hash2-reorg"}

	err = l1BlockOrm.InsertL1Blocks(context.Background(), []L1Block{block1, block2, block3})
	assert.NoError(t, err)

	height, err := l1BlockOrm.GetLatestL1BlockHeight(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), height)

	blocks, err := l1BlockOrm.GetL1Blocks(context.Background(), map[string]interface{}{})
	assert.NoError(t, err)
	assert.Len(t, blocks, 3)
	assert.Equal(t, "hash1", blocks[0].Hash)
	assert.Equal(t, "hash2", blocks[1].Hash)
	assert.Equal(t, "hash3", blocks[2].Hash)

	// reorg handling: insert another block with same height and different hash
	err = l1BlockOrm.InsertL1Blocks(context.Background(), []L1Block{block2AfterReorg})
	assert.NoError(t, err)

	blocks, err = l1BlockOrm.GetL1Blocks(context.Background(), map[string]interface{}{})
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, "hash1", blocks[0].Hash)
	assert.Equal(t, "hash2-reorg", blocks[1].Hash)

	err = l1BlockOrm.UpdateL1GasOracleStatusAndOracleTxHash(context.Background(), "hash1", types.GasOracleImported, "txhash1")
	assert.NoError(t, err)

	updatedBlocks, err := l1BlockOrm.GetL1Blocks(context.Background(), map[string]interface{}{})
	assert.NoError(t, err)
	assert.Len(t, updatedBlocks, 2)
	assert.Equal(t, types.GasOracleImported, types.GasOracleStatus(updatedBlocks[0].GasOracleStatus))
	assert.Equal(t, "txhash1", updatedBlocks[0].OracleTxHash)
}

func TestL2BlockOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	err = l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)

	height, err := l2BlockOrm.GetL2BlocksLatestHeight(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), height)

	blocks, err := l2BlockOrm.GetL2Blocks(context.Background(), map[string]interface{}{}, []string{}, 0)
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, "", blocks[0].ChunkHash)
	assert.Equal(t, "", blocks[1].ChunkHash)

	wrappedBlocks, err := l2BlockOrm.GetL2BlocksInRange(context.Background(), 2, 3)
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, wrappedBlock1, wrappedBlocks[0])
	assert.Equal(t, wrappedBlock2, wrappedBlocks[1])

	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 2, 2, "test hash")
	assert.NoError(t, err)

	blocks, err = l2BlockOrm.GetL2Blocks(context.Background(), map[string]interface{}{}, []string{}, 0)
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, "test hash", blocks[0].ChunkHash)
	assert.Equal(t, "", blocks[1].ChunkHash)
}

func TestChunkOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	dbChunk1, err := chunkOrm.InsertChunk(context.Background(), chunk1)
	assert.NoError(t, err)
	assert.Equal(t, dbChunk1.Hash, chunkHash1.Hex())

	dbChunk2, err := chunkOrm.InsertChunk(context.Background(), chunk2)
	assert.NoError(t, err)
	assert.Equal(t, dbChunk2.Hash, chunkHash2.Hex())

	chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
	assert.NoError(t, err)
	assert.Len(t, chunks, 2)
	assert.Equal(t, chunkHash1.Hex(), chunks[0].Hash)
	assert.Equal(t, chunkHash2.Hex(), chunks[1].Hash)
	assert.Equal(t, "", chunks[0].BatchHash)
	assert.Equal(t, "", chunks[1].BatchHash)

	err = chunkOrm.UpdateProvingStatus(context.Background(), chunkHash1.Hex(), types.ProvingTaskVerified)
	assert.NoError(t, err)
	err = chunkOrm.UpdateProvingStatus(context.Background(), chunkHash2.Hex(), types.ProvingTaskAssigned)
	assert.NoError(t, err)

	chunks, err = chunkOrm.GetChunksInRange(context.Background(), 0, 1)
	assert.NoError(t, err)
	assert.Len(t, chunks, 2)
	assert.Equal(t, chunkHash1.Hex(), chunks[0].Hash)
	assert.Equal(t, chunkHash2.Hex(), chunks[1].Hash)
	assert.Equal(t, types.ProvingTaskVerified, types.ProvingStatus(chunks[0].ProvingStatus))
	assert.Equal(t, types.ProvingTaskAssigned, types.ProvingStatus(chunks[1].ProvingStatus))

	err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 0, "test hash")
	assert.NoError(t, err)
	chunks, err = chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
	assert.NoError(t, err)
	assert.Len(t, chunks, 2)
	assert.Equal(t, chunkHash1.Hex(), chunks[0].Hash)
	assert.Equal(t, chunkHash2.Hex(), chunks[1].Hash)
	assert.Equal(t, "test hash", chunks[0].BatchHash)
	assert.Equal(t, "", chunks[1].BatchHash)
}

func TestBatchOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	batchMeta1 := &types.BatchMeta{
		StartChunkIndex: 0,
		StartChunkHash:  chunkHash1.Hex(),
		EndChunkIndex:   0,
		EndChunkHash:    chunkHash1.Hex(),
	}
	batch1, err := batchOrm.InsertBatch(context.Background(), []*types.Chunk{chunk1}, batchMeta1)
	assert.NoError(t, err)
	hash1 := batch1.Hash

	batch1, err = batchOrm.GetBatchByIndex(context.Background(), 0)
	assert.NoError(t, err)
	batchHeader1, err := types.DecodeBatchHeader(batch1.BatchHeader)
	assert.NoError(t, err)
	batchHash1 := batchHeader1.Hash().Hex()
	assert.Equal(t, hash1, batchHash1)

	batchMeta2 := &types.BatchMeta{
		StartChunkIndex: 1,
		StartChunkHash:  chunkHash2.Hex(),
		EndChunkIndex:   1,
		EndChunkHash:    chunkHash2.Hex(),
	}
	batch2, err := batchOrm.InsertBatch(context.Background(), []*types.Chunk{chunk2}, batchMeta2)
	assert.NoError(t, err)
	hash2 := batch2.Hash

	batch2, err = batchOrm.GetBatchByIndex(context.Background(), 1)
	assert.NoError(t, err)
	batchHeader2, err := types.DecodeBatchHeader(batch2.BatchHeader)
	assert.NoError(t, err)
	batchHash2 := batchHeader2.Hash().Hex()
	assert.Equal(t, hash2, batchHash2)

	count, err := batchOrm.GetBatchCount(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), count)

	err = batchOrm.UpdateRollupStatus(context.Background(), batchHash1, types.RollupCommitFailed)
	assert.NoError(t, err)

	pendingBatches, err := batchOrm.GetFailedAndPendingBatches(context.Background(), 100)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pendingBatches))

	rollupStatus, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batchHash1, batchHash2})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(rollupStatus))
	assert.Equal(t, types.RollupCommitFailed, rollupStatus[0])
	assert.Equal(t, types.RollupPending, rollupStatus[1])

	err = batchOrm.UpdateProvingStatus(context.Background(), batchHash2, types.ProvingTaskVerified)
	assert.NoError(t, err)

	dbProof, err := batchOrm.GetVerifiedProofByHash(context.Background(), batchHash1)
	assert.Error(t, err)
	assert.Nil(t, dbProof)

	err = batchOrm.UpdateProvingStatus(context.Background(), batchHash2, types.ProvingTaskVerified)
	assert.NoError(t, err)
	err = batchOrm.UpdateRollupStatus(context.Background(), batchHash2, types.RollupFinalized)
	assert.NoError(t, err)
	err = batchOrm.UpdateL2GasOracleStatusAndOracleTxHash(context.Background(), batchHash2, types.GasOracleImported, "oracleTxHash")
	assert.NoError(t, err)

	updatedBatch, err := batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, updatedBatch)
	assert.Equal(t, types.ProvingTaskVerified, types.ProvingStatus(updatedBatch.ProvingStatus))
	assert.Equal(t, types.RollupFinalized, types.RollupStatus(updatedBatch.RollupStatus))
	assert.Equal(t, types.GasOracleImported, types.GasOracleStatus(updatedBatch.OracleStatus))
	assert.Equal(t, "oracleTxHash", updatedBatch.OracleTxHash)

	err = batchOrm.UpdateCommitTxHashAndRollupStatus(context.Background(), batchHash2, "commitTxHash", types.RollupCommitted)
	assert.NoError(t, err)
	updatedBatch, err = batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, updatedBatch)
	assert.Equal(t, "commitTxHash", updatedBatch.CommitTxHash)
	assert.Equal(t, types.RollupCommitted, types.RollupStatus(updatedBatch.RollupStatus))

	err = batchOrm.UpdateFinalizeTxHashAndRollupStatus(context.Background(), batchHash2, "finalizeTxHash", types.RollupFinalizeFailed)
	assert.NoError(t, err)

	updatedBatch, err = batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, updatedBatch)
	assert.Equal(t, "finalizeTxHash", updatedBatch.FinalizeTxHash)
	assert.Equal(t, types.RollupFinalizeFailed, types.RollupStatus(updatedBatch.RollupStatus))
}

func TestTransactionOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	tx1 := createTestTransaction(1, big.NewInt(0), big.NewInt(0))
	tx2 := createTestTransaction(1, big.NewInt(1), big.NewInt(1))
	senderMeta := createTestSenderMeta("0xSomeAddress", "TestSender", "TestService", types.SenderTypeUnknown)

	err = pendingTransactionOrm.InsertPendingTransaction(context.Background(), "TestContextID", senderMeta, tx1, 0)
	assert.NoError(t, err)

	err = pendingTransactionOrm.InsertPendingTransaction(context.Background(), "TestContextID", senderMeta, tx2, 0)
	assert.NoError(t, err)

	err = pendingTransactionOrm.UpdatePendingTransactionStatusByTxHash(context.Background(), tx1.Hash().String(), types.TxStatusReplaced)
	assert.NoError(t, err)

	txs, err := pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), senderMeta.Type, 2)
	assert.NoError(t, err)
	assert.Len(t, txs, 2)

	err = pendingTransactionOrm.UpdatePendingTransactionStatusByTxHash(context.Background(), tx2.Hash().String(), types.TxStatusConfirmed)
	assert.NoError(t, err)

	txs, err = pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), senderMeta.Type, 2)
	assert.NoError(t, err)
	assert.Len(t, txs, 1)

	err = pendingTransactionOrm.UpdateOtherTransactionsAsFailedByNonce(context.Background(), senderMeta.Address.String(), tx2.Nonce(), tx2.Hash().String())
	assert.NoError(t, err)

	txs, err = pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), senderMeta.Type, 2)
	assert.NoError(t, err)
	assert.Len(t, txs, 0)

	status, err := pendingTransactionOrm.GetTxStatusByTxHash(context.Background(), tx1.Hash().String())
	assert.NoError(t, err)
	assert.Equal(t, types.TxStatusFailed, status)
}

func createTestTransaction(nonce uint64, gasTipCap, gasFeeCap *big.Int) *gethTypes.Transaction {
	txData := &gethTypes.DynamicFeeTx{
		Nonce:      nonce,
		To:         &common.Address{},
		Data:       []byte{},
		Gas:        21000,
		AccessList: gethTypes.AccessList{},
		Value:      big.NewInt(0),
		ChainID:    big.NewInt(1),
		GasTipCap:  gasTipCap,
		GasFeeCap:  gasFeeCap,
		V:          big.NewInt(0),
		R:          big.NewInt(0),
		S:          big.NewInt(0),
	}
	return gethTypes.NewTx(txData)
}

func createTestSenderMeta(address, name, service string, senderType types.SenderType) *SenderMeta {
	return &SenderMeta{
		Name:    name,
		Service: service,
		Address: common.HexToAddress(address),
		Type:    senderType,
	}
}
