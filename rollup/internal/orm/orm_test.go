package orm

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/da-codec/encoding/codecv0"
	"github.com/scroll-tech/da-codec/encoding/codecv1"
	"github.com/scroll-tech/da-codec/encoding/codecv2"
	"github.com/scroll-tech/da-codec/encoding/codecv3"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/testcontainers"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/database/migrate"

	"scroll-tech/rollup/internal/utils"
)

var (
	testApps *testcontainers.TestcontainerApps

	db                    *gorm.DB
	l2BlockOrm            *L2Block
	chunkOrm              *Chunk
	batchOrm              *Batch
	bundleOrm             *Bundle
	pendingTransactionOrm *PendingTransaction

	block1 *encoding.Block
	block2 *encoding.Block
)

func TestMain(m *testing.M) {
	t := &testing.T{}
	defer func() {
		if testApps != nil {
			testApps.Free()
		}
		tearDownEnv(t)
	}()
	setupEnv(t)
	m.Run()
}

func setupEnv(t *testing.T) {
	var err error

	testApps = testcontainers.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())

	db, err = testApps.GetGormDBClient()
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	bundleOrm = NewBundle(db)
	batchOrm = NewBatch(db)
	chunkOrm = NewChunk(db)
	l2BlockOrm = NewL2Block(db)
	pendingTransactionOrm = NewPendingTransaction(db)

	templateBlockTrace, err := os.ReadFile("../../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	block1 = &encoding.Block{}
	err = json.Unmarshal(templateBlockTrace, block1)
	assert.NoError(t, err)

	templateBlockTrace, err = os.ReadFile("../../../common/testdata/blockTrace_03.json")
	assert.NoError(t, err)
	block2 = &encoding.Block{}
	err = json.Unmarshal(templateBlockTrace, block2)
	assert.NoError(t, err)
}

func tearDownEnv(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	sqlDB.Close()
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

	err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)

	height, err := l2BlockOrm.GetL2BlocksLatestHeight(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), height)

	chunkHashes, err := l2BlockOrm.GetChunkHashes(context.Background(), 0)
	assert.NoError(t, err)
	assert.Len(t, chunkHashes, 2)
	assert.Equal(t, "", chunkHashes[0])
	assert.Equal(t, "", chunkHashes[1])

	blocks, err := l2BlockOrm.GetL2BlocksInRange(context.Background(), 2, 3)
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, block1, blocks[0])
	assert.Equal(t, block2, blocks[1])

	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 2, 2, "test hash")
	assert.NoError(t, err)

	chunkHashes, err = l2BlockOrm.GetChunkHashes(context.Background(), 0)
	assert.NoError(t, err)
	assert.Len(t, chunkHashes, 2)
	assert.Equal(t, "test hash", chunkHashes[0])
	assert.Equal(t, "", chunkHashes[1])
}

func TestChunkOrm(t *testing.T) {
	codecVersions := []encoding.CodecVersion{encoding.CodecV0, encoding.CodecV1, encoding.CodecV2, encoding.CodecV3}
	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}
	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}
	for _, codecVersion := range codecVersions {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))
		var chunkHash1 common.Hash
		var chunkHash2 common.Hash
		if codecVersion == encoding.CodecV0 {
			daChunk1, createErr := codecv0.NewDAChunk(chunk1, 0)
			assert.NoError(t, createErr)
			chunkHash1, err = daChunk1.Hash()
			assert.NoError(t, err)

			daChunk2, createErr := codecv0.NewDAChunk(chunk2, chunk1.NumL1Messages(0))
			assert.NoError(t, createErr)
			chunkHash2, err = daChunk2.Hash()
			assert.NoError(t, err)
		} else if codecVersion == encoding.CodecV1 {
			daChunk1, createErr := codecv1.NewDAChunk(chunk1, 0)
			assert.NoError(t, createErr)
			chunkHash1, err = daChunk1.Hash()
			assert.NoError(t, err)

			daChunk2, createErr := codecv1.NewDAChunk(chunk2, chunk1.NumL1Messages(0))
			assert.NoError(t, createErr)
			chunkHash2, err = daChunk2.Hash()
			assert.NoError(t, err)
		} else if codecVersion == encoding.CodecV2 {
			daChunk1, createErr := codecv2.NewDAChunk(chunk1, 0)
			assert.NoError(t, createErr)
			chunkHash1, err = daChunk1.Hash()
			assert.NoError(t, err)

			daChunk2, createErr := codecv2.NewDAChunk(chunk2, chunk1.NumL1Messages(0))
			assert.NoError(t, createErr)
			chunkHash2, err = daChunk2.Hash()
			assert.NoError(t, err)
		} else {
			daChunk1, createErr := codecv3.NewDAChunk(chunk1, 0)
			assert.NoError(t, createErr)
			chunkHash1, err = daChunk1.Hash()
			assert.NoError(t, err)

			daChunk2, createErr := codecv3.NewDAChunk(chunk2, chunk1.NumL1Messages(0))
			assert.NoError(t, createErr)
			chunkHash2, err = daChunk2.Hash()
			assert.NoError(t, err)
		}

		dbChunk1, err := chunkOrm.InsertChunk(context.Background(), chunk1, utils.CodecConfig{Version: codecVersion}, utils.ChunkMetrics{})
		assert.NoError(t, err)
		assert.Equal(t, dbChunk1.Hash, chunkHash1.Hex())

		dbChunk2, err := chunkOrm.InsertChunk(context.Background(), chunk2, utils.CodecConfig{Version: codecVersion}, utils.ChunkMetrics{})
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
}

func TestBatchOrm(t *testing.T) {
	codecVersions := []encoding.CodecVersion{encoding.CodecV0, encoding.CodecV1, encoding.CodecV2, encoding.CodecV3}
	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}
	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}
	for _, codecVersion := range codecVersions {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		batch := &encoding.Batch{
			Index:  0,
			Chunks: []*encoding.Chunk{chunk1},
		}
		batch1, err := batchOrm.InsertBatch(context.Background(), batch, utils.CodecConfig{Version: codecVersion}, utils.BatchMetrics{})
		assert.NoError(t, err)
		hash1 := batch1.Hash

		batch1, err = batchOrm.GetBatchByIndex(context.Background(), 0)
		assert.NoError(t, err)

		var batchHash1 string
		if codecVersion == encoding.CodecV0 {
			daBatch1, createErr := codecv0.NewDABatchFromBytes(batch1.BatchHeader)
			assert.NoError(t, createErr)
			batchHash1 = daBatch1.Hash().Hex()
		} else if codecVersion == encoding.CodecV1 {
			daBatch1, createErr := codecv1.NewDABatchFromBytes(batch1.BatchHeader)
			assert.NoError(t, createErr)
			batchHash1 = daBatch1.Hash().Hex()
		} else if codecVersion == encoding.CodecV2 {
			daBatch1, createErr := codecv2.NewDABatchFromBytes(batch1.BatchHeader)
			assert.NoError(t, createErr)
			batchHash1 = daBatch1.Hash().Hex()
		} else {
			daBatch1, createErr := codecv3.NewDABatchFromBytes(batch1.BatchHeader)
			assert.NoError(t, createErr)
			batchHash1 = daBatch1.Hash().Hex()
		}
		assert.Equal(t, hash1, batchHash1)

		batch = &encoding.Batch{
			Index:  1,
			Chunks: []*encoding.Chunk{chunk2},
		}
		batch2, err := batchOrm.InsertBatch(context.Background(), batch, utils.CodecConfig{Version: codecVersion}, utils.BatchMetrics{})
		assert.NoError(t, err)
		hash2 := batch2.Hash

		batch2, err = batchOrm.GetBatchByIndex(context.Background(), 1)
		assert.NoError(t, err)

		var batchHash2 string
		if codecVersion == encoding.CodecV0 {
			daBatch2, createErr := codecv0.NewDABatchFromBytes(batch2.BatchHeader)
			assert.NoError(t, createErr)
			batchHash2 = daBatch2.Hash().Hex()
		} else {
			daBatch2, createErr := codecv1.NewDABatchFromBytes(batch2.BatchHeader)
			assert.NoError(t, createErr)
			batchHash2 = daBatch2.Hash().Hex()
		}
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

		batches, err := batchOrm.GetBatchesGEIndexGECodecVersion(context.Background(), 0, codecVersion, 0)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(batches))
		assert.Equal(t, batchHash1, batches[0].Hash)
		assert.Equal(t, batchHash2, batches[1].Hash)

		batches, err = batchOrm.GetBatchesGEIndexGECodecVersion(context.Background(), 0, codecVersion, 1)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(batches))
		assert.Equal(t, batchHash1, batches[0].Hash)

		batches, err = batchOrm.GetBatchesGEIndexGECodecVersion(context.Background(), 1, codecVersion, 0)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(batches))
		assert.Equal(t, batchHash2, batches[0].Hash)

		batches, err = batchOrm.GetBatchesGEIndexGECodecVersion(context.Background(), 0, codecVersion+1, 0)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(batches))

		err = batchOrm.UpdateBundleHashInRange(context.Background(), 0, 0, "test hash")
		assert.NoError(t, err)

		err = batchOrm.UpdateProvingStatusByBundleHash(context.Background(), "test hash", types.ProvingTaskFailed)
		assert.NoError(t, err)

		err = batchOrm.UpdateFinalizeTxHashAndRollupStatusByBundleHash(context.Background(), "test hash", "tx hash", types.RollupCommitFailed)
		assert.NoError(t, err)

		batches, err = batchOrm.GetBatchesGEIndexGECodecVersion(context.Background(), 0, codecVersion, 0)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(batches))
		assert.Equal(t, batchHash1, batches[0].Hash)
		assert.Equal(t, batchHash2, batches[1].Hash)
		assert.Equal(t, types.ProvingTaskFailed, types.ProvingStatus(batches[0].ProvingStatus))
		assert.Equal(t, types.RollupCommitFailed, types.RollupStatus(batches[0].RollupStatus))
		assert.Equal(t, types.ProvingTaskVerified, types.ProvingStatus(batches[1].ProvingStatus))
		assert.Equal(t, types.RollupFinalizeFailed, types.RollupStatus(batches[1].RollupStatus))
	}
}

func TestBundleOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}
	batch1 := &encoding.Batch{
		Index:  0,
		Chunks: []*encoding.Chunk{chunk1},
	}
	dbBatch1, err := batchOrm.InsertBatch(context.Background(), batch1, utils.CodecConfig{Version: encoding.CodecV3}, utils.BatchMetrics{})
	assert.NoError(t, err)

	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}
	batch2 := &encoding.Batch{
		Index:  1,
		Chunks: []*encoding.Chunk{chunk2},
	}
	dbBatch2, err := batchOrm.InsertBatch(context.Background(), batch2, utils.CodecConfig{Version: encoding.CodecV3}, utils.BatchMetrics{})
	assert.NoError(t, err)

	var bundle1 *Bundle
	var bundle2 *Bundle

	t.Run("InsertBundle", func(t *testing.T) {
		bundle1, err = bundleOrm.InsertBundle(context.Background(), []*Batch{dbBatch1}, encoding.CodecV3)
		assert.NoError(t, err)
		assert.NotNil(t, bundle1)
		assert.Equal(t, uint64(0), bundle1.StartBatchIndex)
		assert.Equal(t, uint64(0), bundle1.EndBatchIndex)
		assert.Equal(t, dbBatch1.Hash, bundle1.StartBatchHash)
		assert.Equal(t, dbBatch1.Hash, bundle1.EndBatchHash)
		assert.Equal(t, encoding.CodecV3, encoding.CodecVersion(bundle1.CodecVersion))

		bundle2, err = bundleOrm.InsertBundle(context.Background(), []*Batch{dbBatch2}, encoding.CodecV3)
		assert.NoError(t, err)
		assert.NotNil(t, bundle2)
		assert.Equal(t, uint64(1), bundle2.StartBatchIndex)
		assert.Equal(t, uint64(1), bundle2.EndBatchIndex)
		assert.Equal(t, dbBatch2.Hash, bundle2.StartBatchHash)
		assert.Equal(t, dbBatch2.Hash, bundle2.EndBatchHash)
		assert.Equal(t, encoding.CodecV3, encoding.CodecVersion(bundle2.CodecVersion))
	})

	t.Run("GetFirstUnbundledBatchIndex", func(t *testing.T) {
		index, err := bundleOrm.GetFirstUnbundledBatchIndex(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, uint64(2), index)
	})

	t.Run("GetFirstPendingBundle", func(t *testing.T) {
		bundle, err := bundleOrm.GetFirstPendingBundle(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, bundle)
		assert.Equal(t, int16(types.RollupPending), bundle.RollupStatus)
	})

	t.Run("UpdateFinalizeTxHashAndRollupStatus", func(t *testing.T) {
		err := bundleOrm.UpdateFinalizeTxHashAndRollupStatus(context.Background(), bundle1.Hash, "0xabcd", types.RollupFinalized)
		assert.NoError(t, err)

		pendingBundle, err := bundleOrm.GetFirstPendingBundle(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, uint64(2), pendingBundle.Index)

		var finalizedBundle Bundle
		err = db.Where("hash = ?", bundle1.Hash).First(&finalizedBundle).Error
		assert.NoError(t, err)
		assert.Equal(t, "0xabcd", finalizedBundle.FinalizeTxHash)
		assert.Equal(t, int16(types.RollupFinalized), finalizedBundle.RollupStatus)
		assert.NotNil(t, finalizedBundle.FinalizedAt)
	})

	t.Run("UpdateProvingStatus", func(t *testing.T) {
		err := bundleOrm.UpdateProvingStatus(context.Background(), bundle1.Hash, types.ProvingTaskAssigned)
		assert.NoError(t, err)

		var bundle Bundle
		err = db.Where("hash = ?", bundle1.Hash).First(&bundle).Error
		assert.NoError(t, err)
		assert.Equal(t, int16(types.ProvingTaskAssigned), bundle.ProvingStatus)

		err = bundleOrm.UpdateProvingStatus(context.Background(), bundle1.Hash, types.ProvingTaskVerified)
		assert.NoError(t, err)

		err = db.Where("hash = ?", bundle1.Hash).First(&bundle).Error
		assert.NoError(t, err)
		assert.Equal(t, int16(types.ProvingTaskVerified), bundle.ProvingStatus)
		assert.NotNil(t, bundle.ProvedAt)
	})

	t.Run("GetVerifiedProofByHash", func(t *testing.T) {
		proof := &message.BundleProof{
			Proof: []byte("test proof"),
		}
		proofBytes, err := json.Marshal(proof)
		assert.NoError(t, err)

		err = db.Model(&Bundle{}).Where("hash = ?", bundle1.Hash).Update("proof", proofBytes).Error
		assert.NoError(t, err)

		retrievedProof, err := bundleOrm.GetVerifiedProofByHash(context.Background(), bundle1.Hash)
		assert.NoError(t, err)
		assert.Equal(t, proof.Proof, retrievedProof.Proof)
	})

	t.Run("GetBundles", func(t *testing.T) {
		bundles, err := bundleOrm.GetBundles(context.Background(), map[string]interface{}{}, []string{}, 0)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(bundles))
		assert.Equal(t, bundle1.Hash, bundles[0].Hash)
		assert.Equal(t, bundle2.Hash, bundles[1].Hash)
	})

	t.Run("UpdateProofAndProvingStatusByHash", func(t *testing.T) {
		proof := &message.BundleProof{
			Proof: []byte("new test proof"),
		}
		err := bundleOrm.UpdateProofAndProvingStatusByHash(context.Background(), bundle2.Hash, proof, types.ProvingTaskVerified, 600)
		assert.NoError(t, err)

		var bundle Bundle
		err = db.Where("hash = ?", bundle2.Hash).First(&bundle).Error
		assert.NoError(t, err)
		assert.Equal(t, types.ProvingTaskVerified, types.ProvingStatus(bundle.ProvingStatus))
		assert.Equal(t, int32(600), bundle.ProofTimeSec)
		assert.NotNil(t, bundle.ProvedAt)

		var retrievedProof message.BundleProof
		err = json.Unmarshal(bundle.Proof, &retrievedProof)
		assert.NoError(t, err)
		assert.Equal(t, proof.Proof, retrievedProof.Proof)
	})

	t.Run("UpdateRollupStatus", func(t *testing.T) {
		err := bundleOrm.UpdateRollupStatus(context.Background(), bundle2.Hash, types.RollupFinalized)
		assert.NoError(t, err)

		var bundle Bundle
		err = db.Where("hash = ?", bundle2.Hash).First(&bundle).Error
		assert.NoError(t, err)
		assert.Equal(t, types.RollupFinalized, types.RollupStatus(bundle.RollupStatus))
		assert.NotNil(t, bundle.FinalizedAt)
	})
}

func TestPendingTransactionOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	tx0 := gethTypes.NewTx(&gethTypes.DynamicFeeTx{
		Nonce:      0,
		To:         &common.Address{},
		Data:       []byte{},
		Gas:        21000,
		AccessList: gethTypes.AccessList{},
		Value:      big.NewInt(0),
		ChainID:    big.NewInt(1),
		GasTipCap:  big.NewInt(0),
		GasFeeCap:  big.NewInt(1),
		V:          big.NewInt(0),
		R:          big.NewInt(0),
		S:          big.NewInt(0),
	})
	tx1 := gethTypes.NewTx(&gethTypes.DynamicFeeTx{
		Nonce:      0,
		To:         &common.Address{},
		Data:       []byte{},
		Gas:        42000,
		AccessList: gethTypes.AccessList{},
		Value:      big.NewInt(0),
		ChainID:    big.NewInt(1),
		GasTipCap:  big.NewInt(1),
		GasFeeCap:  big.NewInt(2),
		V:          big.NewInt(0),
		R:          big.NewInt(0),
		S:          big.NewInt(0),
	})
	senderMeta := &SenderMeta{
		Name:    "testName",
		Service: "testService",
		Address: common.HexToAddress("0x1"),
		Type:    types.SenderTypeCommitBatch,
	}

	err = pendingTransactionOrm.InsertPendingTransaction(context.Background(), "test", senderMeta, tx0, 0)
	assert.NoError(t, err)

	err = pendingTransactionOrm.InsertPendingTransaction(context.Background(), "test", senderMeta, tx1, 0)
	assert.NoError(t, err)

	err = pendingTransactionOrm.UpdatePendingTransactionStatusByTxHash(context.Background(), tx0.Hash(), types.TxStatusReplaced)
	assert.NoError(t, err)

	txs, err := pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), senderMeta.Type, 2)
	assert.NoError(t, err)
	assert.Len(t, txs, 2)
	assert.Equal(t, tx1.Type(), txs[1].Type)
	assert.Equal(t, tx1.Nonce(), txs[1].Nonce)
	assert.Equal(t, tx1.Gas(), txs[1].GasLimit)
	assert.Equal(t, tx1.GasTipCap().Uint64(), txs[1].GasTipCap)
	assert.Equal(t, tx1.GasFeeCap().Uint64(), txs[1].GasFeeCap)
	assert.Equal(t, tx1.ChainId().Uint64(), txs[1].ChainID)
	assert.Equal(t, senderMeta.Name, txs[1].SenderName)
	assert.Equal(t, senderMeta.Service, txs[1].SenderService)
	assert.Equal(t, senderMeta.Address.String(), txs[1].SenderAddress)
	assert.Equal(t, senderMeta.Type, txs[1].SenderType)

	err = pendingTransactionOrm.UpdatePendingTransactionStatusByTxHash(context.Background(), tx1.Hash(), types.TxStatusConfirmed)
	assert.NoError(t, err)

	txs, err = pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), senderMeta.Type, 2)
	assert.NoError(t, err)
	assert.Len(t, txs, 1)

	err = pendingTransactionOrm.UpdateOtherTransactionsAsFailedByNonce(context.Background(), senderMeta.Address.String(), tx1.Nonce(), tx1.Hash())
	assert.NoError(t, err)

	txs, err = pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), senderMeta.Type, 2)
	assert.NoError(t, err)
	assert.Len(t, txs, 0)

	status, err := pendingTransactionOrm.GetTxStatusByTxHash(context.Background(), tx0.Hash())
	assert.NoError(t, err)
	assert.Equal(t, types.TxStatusConfirmedFailed, status)
}
