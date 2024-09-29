package relayer

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/database/migrate"

	"scroll-tech/rollup/internal/controller/sender"
	"scroll-tech/rollup/internal/orm"
	rutils "scroll-tech/rollup/internal/utils"
)

func setupL2RelayerDB(t *testing.T) *gorm.DB {
	db, err := database.InitDB(cfg.DBConfig)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	return db
}

func testCreateNewRelayer(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig, &params.ChainConfig{}, true, ServiceTypeL2RollupRelayer, nil)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)
	defer relayer.StopSenders()
}

func testL2RelayerProcessPendingBatches(t *testing.T) {
	codecVersions := []encoding.CodecVersion{encoding.CodecV0, encoding.CodecV1, encoding.CodecV2, encoding.CodecV3}
	for _, codecVersion := range codecVersions {
		db := setupL2RelayerDB(t)
		defer database.CloseDB(db)

		l2Cfg := cfg.L2Config
		var chainConfig *params.ChainConfig
		if codecVersion == encoding.CodecV0 {
			chainConfig = &params.ChainConfig{}
		} else if codecVersion == encoding.CodecV1 {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0)}
		} else if codecVersion == encoding.CodecV2 {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0)}
		} else {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64)}
		}

		relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig, chainConfig, true, ServiceTypeL2RollupRelayer, nil)
		assert.NoError(t, err)

		patchGuard := gomonkey.ApplyMethodFunc(l2Cli, "SendTransaction", func(_ context.Context, _ *gethTypes.Transaction) error {
			return nil
		})

		l2BlockOrm := orm.NewL2Block(db)
		err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
		assert.NoError(t, err)
		chunkOrm := orm.NewChunk(db)
		_, err = chunkOrm.InsertChunk(context.Background(), chunk1, rutils.CodecConfig{Version: codecVersion}, rutils.ChunkMetrics{})
		assert.NoError(t, err)
		_, err = chunkOrm.InsertChunk(context.Background(), chunk2, rutils.CodecConfig{Version: codecVersion}, rutils.ChunkMetrics{})
		assert.NoError(t, err)

		batch := &encoding.Batch{
			Index:                      1,
			TotalL1MessagePoppedBefore: 0,
			ParentBatchHash:            common.Hash{},
			Chunks:                     []*encoding.Chunk{chunk1, chunk2},
		}

		batchOrm := orm.NewBatch(db)
		dbBatch, err := batchOrm.InsertBatch(context.Background(), batch, rutils.CodecConfig{Version: codecVersion}, rutils.BatchMetrics{})
		assert.NoError(t, err)

		relayer.ProcessPendingBatches()

		statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{dbBatch.Hash})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(statuses))
		assert.Equal(t, types.RollupCommitting, statuses[0])
		relayer.StopSenders()
		patchGuard.Reset()
	}
}

func testL2RelayerProcessCommittedBatches(t *testing.T) {
	codecVersions := []encoding.CodecVersion{encoding.CodecV0, encoding.CodecV1, encoding.CodecV2}
	for _, codecVersion := range codecVersions {
		db := setupL2RelayerDB(t)
		defer database.CloseDB(db)

		l2Cfg := cfg.L2Config
		var chainConfig *params.ChainConfig
		if codecVersion == encoding.CodecV0 {
			chainConfig = &params.ChainConfig{}
		} else if codecVersion == encoding.CodecV1 {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0)}
		} else {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0)}
		}
		relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig, chainConfig, true, ServiceTypeL2RollupRelayer, nil)
		assert.NoError(t, err)

		l2BlockOrm := orm.NewL2Block(db)
		err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
		assert.NoError(t, err)
		chunkOrm := orm.NewChunk(db)
		_, err = chunkOrm.InsertChunk(context.Background(), chunk1, rutils.CodecConfig{Version: codecVersion}, rutils.ChunkMetrics{})
		assert.NoError(t, err)
		_, err = chunkOrm.InsertChunk(context.Background(), chunk2, rutils.CodecConfig{Version: codecVersion}, rutils.ChunkMetrics{})
		assert.NoError(t, err)

		batch := &encoding.Batch{
			Index:                      1,
			TotalL1MessagePoppedBefore: 0,
			ParentBatchHash:            common.Hash{},
			Chunks:                     []*encoding.Chunk{chunk1, chunk2},
		}

		batchOrm := orm.NewBatch(db)
		dbBatch, err := batchOrm.InsertBatch(context.Background(), batch, rutils.CodecConfig{Version: codecVersion}, rutils.BatchMetrics{})
		assert.NoError(t, err)

		err = batchOrm.UpdateRollupStatus(context.Background(), dbBatch.Hash, types.RollupCommitted)
		assert.NoError(t, err)

		err = batchOrm.UpdateProvingStatus(context.Background(), dbBatch.Hash, types.ProvingTaskVerified)
		assert.NoError(t, err)

		relayer.ProcessCommittedBatches()

		statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{dbBatch.Hash})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(statuses))
		// no valid proof, rollup status remains the same
		assert.Equal(t, types.RollupCommitted, statuses[0])

		proof := &message.BatchProof{
			Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Instances: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Vk:        []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		}
		err = batchOrm.UpdateProofByHash(context.Background(), dbBatch.Hash, proof, 100)
		assert.NoError(t, err)

		relayer.ProcessCommittedBatches()
		statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{dbBatch.Hash})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(statuses))
		assert.Equal(t, types.RollupFinalizing, statuses[0])
		relayer.StopSenders()
	}
}

func testL2RelayerProcessPendingBundles(t *testing.T) {
	codecVersions := []encoding.CodecVersion{encoding.CodecV3}
	for _, codecVersion := range codecVersions {
		db := setupL2RelayerDB(t)
		defer database.CloseDB(db)

		l2Cfg := cfg.L2Config
		var chainConfig *params.ChainConfig
		if codecVersion == encoding.CodecV3 {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64)}
		}
		relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig, chainConfig, true, ServiceTypeL2RollupRelayer, nil)
		assert.NoError(t, err)

		batch := &encoding.Batch{
			Index:                      1,
			TotalL1MessagePoppedBefore: 0,
			ParentBatchHash:            common.Hash{},
			Chunks:                     []*encoding.Chunk{chunk1, chunk2},
		}

		batchOrm := orm.NewBatch(db)
		dbBatch, err := batchOrm.InsertBatch(context.Background(), batch, rutils.CodecConfig{Version: codecVersion}, rutils.BatchMetrics{})
		assert.NoError(t, err)

		bundleOrm := orm.NewBundle(db)
		bundle, err := bundleOrm.InsertBundle(context.Background(), []*orm.Batch{dbBatch}, codecVersion)
		assert.NoError(t, err)

		err = bundleOrm.UpdateRollupStatus(context.Background(), bundle.Hash, types.RollupPending)
		assert.NoError(t, err)

		err = bundleOrm.UpdateProvingStatus(context.Background(), dbBatch.Hash, types.ProvingTaskVerified)
		assert.NoError(t, err)

		relayer.ProcessPendingBundles()

		bundles, err := bundleOrm.GetBundles(context.Background(), map[string]interface{}{"hash": bundle.Hash}, nil, 0)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(bundles))
		// no valid proof, rollup status remains the same
		assert.Equal(t, types.RollupPending, types.RollupStatus(bundles[0].RollupStatus))

		proof := &message.BundleProof{
			Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Instances: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Vk:        []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		}
		err = bundleOrm.UpdateProofAndProvingStatusByHash(context.Background(), bundle.Hash, proof, types.ProvingTaskVerified, 600)
		assert.NoError(t, err)

		relayer.ProcessPendingBundles()
		bundles, err = bundleOrm.GetBundles(context.Background(), map[string]interface{}{"hash": bundle.Hash}, nil, 0)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(bundles))
		assert.Equal(t, types.RollupFinalizing, types.RollupStatus(bundles[0].RollupStatus))
		relayer.StopSenders()
	}
}

func testL2RelayerFinalizeTimeoutBatches(t *testing.T) {
	codecVersions := []encoding.CodecVersion{encoding.CodecV0, encoding.CodecV1, encoding.CodecV2}
	for _, codecVersion := range codecVersions {
		db := setupL2RelayerDB(t)
		defer database.CloseDB(db)

		l2Cfg := cfg.L2Config
		l2Cfg.RelayerConfig.EnableTestEnvBypassFeatures = true
		l2Cfg.RelayerConfig.FinalizeBatchWithoutProofTimeoutSec = 0
		var chainConfig *params.ChainConfig
		if codecVersion == encoding.CodecV0 {
			chainConfig = &params.ChainConfig{}
		} else if codecVersion == encoding.CodecV1 {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0)}
		} else {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0)}
		}
		relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig, chainConfig, true, ServiceTypeL2RollupRelayer, nil)
		assert.NoError(t, err)

		l2BlockOrm := orm.NewL2Block(db)
		err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
		assert.NoError(t, err)
		chunkOrm := orm.NewChunk(db)
		chunkDB1, err := chunkOrm.InsertChunk(context.Background(), chunk1, rutils.CodecConfig{Version: codecVersion}, rutils.ChunkMetrics{})
		assert.NoError(t, err)
		chunkDB2, err := chunkOrm.InsertChunk(context.Background(), chunk2, rutils.CodecConfig{Version: codecVersion}, rutils.ChunkMetrics{})
		assert.NoError(t, err)

		batch := &encoding.Batch{
			Index:                      1,
			TotalL1MessagePoppedBefore: 0,
			ParentBatchHash:            common.Hash{},
			Chunks:                     []*encoding.Chunk{chunk1, chunk2},
		}

		batchOrm := orm.NewBatch(db)
		dbBatch, err := batchOrm.InsertBatch(context.Background(), batch, rutils.CodecConfig{Version: codecVersion}, rutils.BatchMetrics{})
		assert.NoError(t, err)

		err = batchOrm.UpdateRollupStatus(context.Background(), dbBatch.Hash, types.RollupCommitted)
		assert.NoError(t, err)

		err = chunkOrm.UpdateBatchHashInRange(context.Background(), chunkDB1.Index, chunkDB2.Index, dbBatch.Hash, nil)
		assert.NoError(t, err)

		assert.Eventually(t, func() bool {
			relayer.ProcessCommittedBatches()

			batchInDB, batchErr := batchOrm.GetBatches(context.Background(), map[string]interface{}{"hash": dbBatch.Hash}, nil, 0)
			if batchErr != nil {
				return false
			}

			batchStatus := len(batchInDB) == 1 && types.RollupStatus(batchInDB[0].RollupStatus) == types.RollupFinalizing &&
				types.ProvingStatus(batchInDB[0].ProvingStatus) == types.ProvingTaskVerified

			chunks, chunkErr := chunkOrm.GetChunksByBatchHash(context.Background(), dbBatch.Hash)
			if chunkErr != nil {
				return false
			}

			chunkStatus := len(chunks) == 2 && types.ProvingStatus(chunks[0].ProvingStatus) == types.ProvingTaskVerified &&
				types.ProvingStatus(chunks[1].ProvingStatus) == types.ProvingTaskVerified

			return batchStatus && chunkStatus
		}, 5*time.Second, 100*time.Millisecond, "Batch or Chunk status did not update as expected")
		relayer.StopSenders()
	}
}

func testL2RelayerFinalizeTimeoutBundles(t *testing.T) {
	codecVersions := []encoding.CodecVersion{encoding.CodecV3}
	for _, codecVersion := range codecVersions {
		db := setupL2RelayerDB(t)
		defer database.CloseDB(db)

		l2Cfg := cfg.L2Config
		l2Cfg.RelayerConfig.EnableTestEnvBypassFeatures = true
		l2Cfg.RelayerConfig.FinalizeBundleWithoutProofTimeoutSec = 0
		var chainConfig *params.ChainConfig
		if codecVersion == encoding.CodecV3 {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64)}
		}
		relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig, chainConfig, true, ServiceTypeL2RollupRelayer, nil)
		assert.NoError(t, err)

		l2BlockOrm := orm.NewL2Block(db)
		err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
		assert.NoError(t, err)
		chunkOrm := orm.NewChunk(db)
		chunkDB1, err := chunkOrm.InsertChunk(context.Background(), chunk1, rutils.CodecConfig{Version: codecVersion}, rutils.ChunkMetrics{})
		assert.NoError(t, err)
		chunkDB2, err := chunkOrm.InsertChunk(context.Background(), chunk2, rutils.CodecConfig{Version: codecVersion}, rutils.ChunkMetrics{})
		assert.NoError(t, err)

		batch := &encoding.Batch{
			Index:                      1,
			TotalL1MessagePoppedBefore: 0,
			ParentBatchHash:            common.Hash{},
			Chunks:                     []*encoding.Chunk{chunk1, chunk2},
		}

		batchOrm := orm.NewBatch(db)
		dbBatch, err := batchOrm.InsertBatch(context.Background(), batch, rutils.CodecConfig{Version: codecVersion}, rutils.BatchMetrics{})
		assert.NoError(t, err)

		err = batchOrm.UpdateRollupStatus(context.Background(), dbBatch.Hash, types.RollupCommitted)
		assert.NoError(t, err)

		err = chunkOrm.UpdateBatchHashInRange(context.Background(), chunkDB1.Index, chunkDB2.Index, dbBatch.Hash, nil)
		assert.NoError(t, err)

		bundleOrm := orm.NewBundle(db)
		bundle, err := bundleOrm.InsertBundle(context.Background(), []*orm.Batch{dbBatch}, codecVersion)
		assert.NoError(t, err)

		err = batchOrm.UpdateBundleHashInRange(context.Background(), dbBatch.Index, dbBatch.Index, bundle.Hash, nil)
		assert.NoError(t, err)

		assert.Eventually(t, func() bool {
			relayer.ProcessPendingBundles()

			bundleInDB, bundleErr := bundleOrm.GetBundles(context.Background(), map[string]interface{}{"hash": bundle.Hash}, nil, 0)
			if bundleErr != nil {
				return false
			}

			bundleStatus := len(bundleInDB) == 1 && types.RollupStatus(bundleInDB[0].RollupStatus) == types.RollupFinalizing &&
				types.ProvingStatus(bundleInDB[0].ProvingStatus) == types.ProvingTaskVerified

			batchInDB, batchErr := batchOrm.GetBatches(context.Background(), map[string]interface{}{"hash": dbBatch.Hash}, nil, 0)
			if batchErr != nil {
				return false
			}

			batchStatus := len(batchInDB) == 1 && types.ProvingStatus(batchInDB[0].ProvingStatus) == types.ProvingTaskVerified

			chunks, chunkErr := chunkOrm.GetChunksByBatchHash(context.Background(), dbBatch.Hash)
			if chunkErr != nil {
				return false
			}

			chunkStatus := len(chunks) == 2 && types.ProvingStatus(chunks[0].ProvingStatus) == types.ProvingTaskVerified &&
				types.ProvingStatus(chunks[1].ProvingStatus) == types.ProvingTaskVerified

			return bundleStatus && batchStatus && chunkStatus
		}, 5*time.Second, 100*time.Millisecond, "Bundle or Batch or Chunk status did not update as expected")
		relayer.StopSenders()
	}
}

func testL2RelayerCommitConfirm(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig, &params.ChainConfig{}, true, ServiceTypeL2RollupRelayer, nil)
	assert.NoError(t, err)
	defer l2Relayer.StopSenders()

	// Simulate message confirmations.
	isSuccessful := []bool{true, false}
	batchOrm := orm.NewBatch(db)
	batchHashes := make([]string, len(isSuccessful))
	for i := range batchHashes {
		batch := &encoding.Batch{
			Index:                      uint64(i + 1),
			TotalL1MessagePoppedBefore: 0,
			ParentBatchHash:            common.Hash{},
			Chunks:                     []*encoding.Chunk{chunk1, chunk2},
		}

		dbBatch, err := batchOrm.InsertBatch(context.Background(), batch, rutils.CodecConfig{Version: encoding.CodecV0}, rutils.BatchMetrics{})
		assert.NoError(t, err)
		batchHashes[i] = dbBatch.Hash
	}

	for i, batchHash := range batchHashes {
		l2Relayer.commitSender.SendConfirmation(&sender.Confirmation{
			ContextID:    batchHash,
			IsSuccessful: isSuccessful[i],
			TxHash:       common.HexToHash("0x123456789abcdef"),
			SenderType:   types.SenderTypeCommitBatch,
		})
	}

	// Check the database for the updated status using TryTimes.
	ok := utils.TryTimes(5, func() bool {
		expectedStatuses := []types.RollupStatus{
			types.RollupCommitted,
			types.RollupCommitFailed,
		}

		for i, batchHash := range batchHashes {
			batchInDB, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{"hash": batchHash}, nil, 0)
			if err != nil || len(batchInDB) != 1 || types.RollupStatus(batchInDB[0].RollupStatus) != expectedStatuses[i] {
				return false
			}
		}
		return true
	})
	assert.True(t, ok)
}

func testL2RelayerFinalizeBatchConfirm(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig, &params.ChainConfig{}, true, ServiceTypeL2RollupRelayer, nil)
	assert.NoError(t, err)
	defer l2Relayer.StopSenders()

	// Simulate message confirmations.
	isSuccessful := []bool{true, false}
	batchOrm := orm.NewBatch(db)
	batchHashes := make([]string, len(isSuccessful))
	for i := range batchHashes {
		batch := &encoding.Batch{
			Index:                      uint64(i + 1),
			TotalL1MessagePoppedBefore: 0,
			ParentBatchHash:            common.Hash{},
			Chunks:                     []*encoding.Chunk{chunk1, chunk2},
		}

		dbBatch, err := batchOrm.InsertBatch(context.Background(), batch, rutils.CodecConfig{Version: encoding.CodecV0}, rutils.BatchMetrics{})
		assert.NoError(t, err)
		batchHashes[i] = dbBatch.Hash
	}

	for i, batchHash := range batchHashes {
		l2Relayer.finalizeSender.SendConfirmation(&sender.Confirmation{
			ContextID:    batchHash,
			IsSuccessful: isSuccessful[i],
			TxHash:       common.HexToHash("0x123456789abcdef"),
			SenderType:   types.SenderTypeFinalizeBatch,
		})
	}

	// Check the database for the updated status using TryTimes.
	ok := utils.TryTimes(5, func() bool {
		expectedStatuses := []types.RollupStatus{
			types.RollupFinalized,
			types.RollupFinalizeFailed,
		}

		for i, batchHash := range batchHashes {
			batchInDB, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{"hash": batchHash}, nil, 0)
			if err != nil || len(batchInDB) != 1 || types.RollupStatus(batchInDB[0].RollupStatus) != expectedStatuses[i] {
				return false
			}
		}
		return true
	})
	assert.True(t, ok)
}

func testL2RelayerFinalizeBundleConfirm(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig, &params.ChainConfig{}, true, ServiceTypeL2RollupRelayer, nil)
	assert.NoError(t, err)
	defer l2Relayer.StopSenders()

	// Simulate message confirmations.
	isSuccessful := []bool{true, false}
	batchOrm := orm.NewBatch(db)
	bundleOrm := orm.NewBundle(db)
	batchHashes := make([]string, len(isSuccessful))
	bundleHashes := make([]string, len(isSuccessful))
	for i := range batchHashes {
		batch := &encoding.Batch{
			Index:                      uint64(i + 1),
			TotalL1MessagePoppedBefore: 0,
			ParentBatchHash:            common.Hash{},
			Chunks:                     []*encoding.Chunk{chunk1, chunk2},
		}

		dbBatch, err := batchOrm.InsertBatch(context.Background(), batch, rutils.CodecConfig{Version: encoding.CodecV0}, rutils.BatchMetrics{})
		assert.NoError(t, err)
		batchHashes[i] = dbBatch.Hash

		bundle, err := bundleOrm.InsertBundle(context.Background(), []*orm.Batch{dbBatch}, encoding.CodecV3)
		assert.NoError(t, err)
		bundleHashes[i] = bundle.Hash

		err = batchOrm.UpdateBundleHashInRange(context.Background(), dbBatch.Index, dbBatch.Index, bundle.Hash)
		assert.NoError(t, err)
	}

	for i, bundleHash := range bundleHashes {
		l2Relayer.finalizeSender.SendConfirmation(&sender.Confirmation{
			ContextID:    "finalizeBundle-" + bundleHash,
			IsSuccessful: isSuccessful[i],
			TxHash:       common.HexToHash("0x123456789abcdef"),
			SenderType:   types.SenderTypeFinalizeBatch,
		})
	}

	assert.Eventually(t, func() bool {
		expectedStatuses := []types.RollupStatus{
			types.RollupFinalized,
			types.RollupFinalizeFailed,
		}

		for i, bundleHash := range bundleHashes {
			bundleInDB, err := bundleOrm.GetBundles(context.Background(), map[string]interface{}{"hash": bundleHash}, nil, 0)
			if err != nil || len(bundleInDB) != 1 || types.RollupStatus(bundleInDB[0].RollupStatus) != expectedStatuses[i] {
				return false
			}

			batchInDB, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{"hash": batchHashes[i]}, nil, 0)
			if err != nil || len(batchInDB) != 1 || types.RollupStatus(batchInDB[0].RollupStatus) != expectedStatuses[i] {
				return false
			}
		}

		return true
	}, 5*time.Second, 100*time.Millisecond, "Bundle or Batch status did not update as expected")
}

func testL2RelayerGasOracleConfirm(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)

	batch1 := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk1},
	}

	batchOrm := orm.NewBatch(db)
	dbBatch1, err := batchOrm.InsertBatch(context.Background(), batch1, rutils.CodecConfig{Version: encoding.CodecV0}, rutils.BatchMetrics{})
	assert.NoError(t, err)

	batch2 := &encoding.Batch{
		Index:                      batch1.Index + 1,
		TotalL1MessagePoppedBefore: batch1.TotalL1MessagePoppedBefore,
		ParentBatchHash:            common.HexToHash(dbBatch1.Hash),
		Chunks:                     []*encoding.Chunk{chunk2},
	}

	dbBatch2, err := batchOrm.InsertBatch(context.Background(), batch2, rutils.CodecConfig{Version: encoding.CodecV0}, rutils.BatchMetrics{})
	assert.NoError(t, err)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig, &params.ChainConfig{}, false, ServiceTypeL2GasOracle, nil)
	assert.NoError(t, err)
	defer l2Relayer.StopSenders()

	// Simulate message confirmations.
	type BatchConfirmation struct {
		batchHash    string
		isSuccessful bool
	}

	confirmations := []BatchConfirmation{
		{batchHash: dbBatch1.Hash, isSuccessful: true},
		{batchHash: dbBatch2.Hash, isSuccessful: false},
	}

	for _, confirmation := range confirmations {
		l2Relayer.gasOracleSender.SendConfirmation(&sender.Confirmation{
			ContextID:    confirmation.batchHash,
			IsSuccessful: confirmation.isSuccessful,
			SenderType:   types.SenderTypeL2GasOracle,
		})
	}
	// Check the database for the updated status using TryTimes.
	ok := utils.TryTimes(5, func() bool {
		expectedStatuses := []types.GasOracleStatus{types.GasOracleImported, types.GasOracleImportedFailed}
		for i, confirmation := range confirmations {
			gasOracle, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{"hash": confirmation.batchHash}, nil, 0)
			if err != nil || len(gasOracle) != 1 || types.GasOracleStatus(gasOracle[0].OracleStatus) != expectedStatuses[i] {
				return false
			}
		}
		return true
	})
	assert.True(t, ok)
}

func testLayer2RelayerProcessGasPriceOracle(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)

	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig, &params.ChainConfig{}, false, ServiceTypeL2GasOracle, nil)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)
	defer relayer.StopSenders()

	var batchOrm *orm.Batch
	convey.Convey("Failed to GetLatestBatch", t, func() {
		targetErr := errors.New("GetLatestBatch error")
		patchGuard := gomonkey.ApplyMethodFunc(batchOrm, "GetLatestBatch", func(context.Context) (*orm.Batch, error) {
			return nil, targetErr
		})
		defer patchGuard.Reset()
		relayer.ProcessGasPriceOracle()
	})

	patchGuard := gomonkey.ApplyMethodFunc(batchOrm, "GetLatestBatch", func(context.Context) (*orm.Batch, error) {
		batch := orm.Batch{
			OracleStatus: int16(types.GasOraclePending),
			Hash:         "0x0000000000000000000000000000000000000000",
		}
		return &batch, nil
	})
	defer patchGuard.Reset()

	convey.Convey("Failed to fetch SuggestGasPrice from l2geth", t, func() {
		targetErr := errors.New("SuggestGasPrice error")
		patchGuard.ApplyMethodFunc(relayer.l2Client, "SuggestGasPrice", func(ctx context.Context) (*big.Int, error) {
			return nil, targetErr
		})
		relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(relayer.l2Client, "SuggestGasPrice", func(ctx context.Context) (*big.Int, error) {
		return big.NewInt(100), nil
	})

	convey.Convey("Failed to pack setL2BaseFee", t, func() {
		targetErr := errors.New("setL2BaseFee error")
		patchGuard.ApplyMethodFunc(relayer.l2GasOracleABI, "Pack", func(name string, args ...interface{}) ([]byte, error) {
			return nil, targetErr
		})
		relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(relayer.l2GasOracleABI, "Pack", func(name string, args ...interface{}) ([]byte, error) {
		return nil, nil
	})

	convey.Convey("Failed to send setL2BaseFee tx to layer2", t, func() {
		targetErr := errors.New("failed to send setL2BaseFee tx to layer2 error")
		patchGuard.ApplyMethodFunc(relayer.gasOracleSender, "SendTransaction", func(ContextID string, target *common.Address, data []byte, blob *kzg4844.Blob, fallbackGasLimit uint64) (hash common.Hash, err error) {
			return common.Hash{}, targetErr
		})
		relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(relayer.gasOracleSender, "SendTransaction", func(ContextID string, target *common.Address, data []byte, blob *kzg4844.Blob, fallbackGasLimit uint64) (hash common.Hash, err error) {
		return common.HexToHash("0x56789abcdef1234"), nil
	})

	convey.Convey("UpdateGasOracleStatusAndOracleTxHash failed", t, func() {
		targetErr := errors.New("UpdateL2GasOracleStatusAndOracleTxHash error")
		patchGuard.ApplyMethodFunc(batchOrm, "UpdateL2GasOracleStatusAndOracleTxHash", func(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error {
			return targetErr
		})
		relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(batchOrm, "UpdateL2GasOracleStatusAndOracleTxHash", func(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error {
		return nil
	})
	relayer.ProcessGasPriceOracle()
}

func mockChainMonitorServer(baseURL string) (*http.Server, error) {
	router := gin.New()
	r := router.Group("/v1")
	r.GET("/batch_status", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
			Data    bool   `json:"data"`
		}{
			ErrCode: 0,
			ErrMsg:  "",
			Data:    true,
		})
	})
	return utils.StartHTTPServer(strings.Split(baseURL, "//")[1], router)
}

func testGetBatchStatusByIndex(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)

	cfg.L2Config.RelayerConfig.ChainMonitor.Enabled = true
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig, &params.ChainConfig{}, true, ServiceTypeL2RollupRelayer, nil)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)
	defer relayer.StopSenders()

	l2BlockOrm := orm.NewL2Block(db)
	err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	chunkOrm := orm.NewChunk(db)
	_, err = chunkOrm.InsertChunk(context.Background(), chunk1, rutils.CodecConfig{Version: encoding.CodecV0}, rutils.ChunkMetrics{})
	assert.NoError(t, err)
	_, err = chunkOrm.InsertChunk(context.Background(), chunk2, rutils.CodecConfig{Version: encoding.CodecV0}, rutils.ChunkMetrics{})
	assert.NoError(t, err)

	batch := &encoding.Batch{
		Index:                      1,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk1, chunk2},
	}

	batchOrm := orm.NewBatch(db)
	dbBatch, err := batchOrm.InsertBatch(context.Background(), batch, rutils.CodecConfig{Version: encoding.CodecV0}, rutils.BatchMetrics{})
	assert.NoError(t, err)

	status, err := relayer.getBatchStatusByIndex(dbBatch)
	assert.NoError(t, err)
	assert.Equal(t, true, status)
}
