package tests

import (
	"context"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/relayer"
	"scroll-tech/rollup/internal/controller/watcher"
	"scroll-tech/rollup/internal/orm"
)

func testCommitAndFinalizeGenesisBatch(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	prepareContracts(t)

	l2Cfg := rollupApp.Config.L2Config
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, &params.ChainConfig{}, true, relayer.ServiceTypeL2RollupRelayer, nil)
	assert.NoError(t, err)
	assert.NotNil(t, l2Relayer)
	defer l2Relayer.StopSenders()

	genesisChunkHash := common.HexToHash("0x00e076380b00a3749816fcc9a2a576b529952ef463222a54544d21b7d434dfe1")
	chunkOrm := orm.NewChunk(db)
	dbChunk, err := chunkOrm.GetChunksInRange(context.Background(), 0, 0)
	assert.NoError(t, err)
	assert.Len(t, dbChunk, 1)
	assert.Equal(t, genesisChunkHash.String(), dbChunk[0].Hash)
	assert.Equal(t, types.ProvingTaskVerified, types.ProvingStatus(dbChunk[0].ProvingStatus))

	genesisBatchHash := common.HexToHash("0x2d214b024f5337d83a5681f88575ab225f345ec2e4e3ce53cf4dc4b0cb5c96b1")
	batchOrm := orm.NewBatch(db)
	batch, err := batchOrm.GetBatchByIndex(context.Background(), 0)
	assert.NoError(t, err)
	assert.Equal(t, genesisBatchHash.String(), batch.Hash)
	assert.Equal(t, types.ProvingTaskVerified, types.ProvingStatus(batch.ProvingStatus))
	assert.Equal(t, types.RollupFinalized, types.RollupStatus(batch.RollupStatus))
}

func testCommitBatchAndFinalizeBundleCodecV4(t *testing.T) {
	codecVersions := []encoding.CodecVersion{encoding.CodecV4}
	for _, codecVersion := range codecVersions {
		db := setupDB(t)

		prepareContracts(t)

		var chainConfig *params.ChainConfig
		if codecVersion == encoding.CodecV4 {
			chainConfig = &params.ChainConfig{LondonBlock: big.NewInt(0), BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64), DarwinV2Time: new(uint64)}
		} else {
			assert.Fail(t, "unsupported codec version, expected CodecV4")
		}

		// Create L2Relayer
		l2Cfg := rollupApp.Config.L2Config
		l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, chainConfig, true, relayer.ServiceTypeL2RollupRelayer, nil)
		assert.NoError(t, err)

		// add some blocks to db
		var blocks []*encoding.Block
		for i := int64(0); i < 10; i++ {
			header := gethTypes.Header{
				Number:     big.NewInt(i + 1),
				ParentHash: common.Hash{},
				Difficulty: big.NewInt(0),
				BaseFee:    big.NewInt(0),
				Root:       common.HexToHash("0x1"),
			}
			blocks = append(blocks, &encoding.Block{
				Header:         &header,
				Transactions:   nil,
				WithdrawRoot:   common.HexToHash("0x2"),
				RowConsumption: &gethTypes.RowConsumption{},
			})
		}

		cp := watcher.NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
			MaxBlockNumPerChunk:             100,
			MaxTxNumPerChunk:                10000,
			MaxL1CommitGasPerChunk:          50000000000,
			MaxL1CommitCalldataSizePerChunk: 1000000,
			MaxRowConsumptionPerChunk:       1048319,
			ChunkTimeoutSec:                 300,
			MaxUncompressedBatchBytesSize:   math.MaxUint64,
		}, encoding.CodecV4, chainConfig, db, nil)

		bap := watcher.NewBatchProposer(context.Background(), &config.BatchProposerConfig{
			MaxL1CommitGasPerBatch:          50000000000,
			MaxL1CommitCalldataSizePerBatch: 1000000,
			BatchTimeoutSec:                 300,
			MaxUncompressedBatchBytesSize:   math.MaxUint64,
		}, encoding.CodecV4, chainConfig, db, nil)

		bup := watcher.NewBundleProposer(context.Background(), &config.BundleProposerConfig{
			MaxBatchNumPerBundle: 1000000,
			BundleTimeoutSec:     300,
		}, encoding.CodecV4, chainConfig, db, nil)

		l2BlockOrm := orm.NewL2Block(db)
		err = l2BlockOrm.InsertL2Blocks(context.Background(), blocks[:5])
		assert.NoError(t, err)

		cp.TryProposeChunk()
		bap.TryProposeBatch()

		err = l2BlockOrm.InsertL2Blocks(context.Background(), blocks[5:])
		assert.NoError(t, err)

		cp.TryProposeChunk()
		bap.TryProposeBatch()

		bup.TryProposeBundle() // The proposed bundle contains two batches when codec version is codecv3.

		l2Relayer.ProcessPendingBatches()

		batchOrm := orm.NewBatch(db)
		bundleOrm := orm.NewBundle(db)

		assert.Eventually(t, func() bool {
			batches, getErr := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, nil, 0)
			assert.NoError(t, getErr)
			assert.Len(t, batches, 3)
			batches = batches[1:]
			for _, batch := range batches {
				if types.RollupCommitted != types.RollupStatus(batch.RollupStatus) {
					return false
				}
			}
			return true
		}, 30*time.Second, time.Second)

		batchProof := &message.BatchProof{
			Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Instances: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Vk:        []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		}
		batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, nil, 0)
		assert.NoError(t, err)
		batches = batches[1:]
		for _, batch := range batches {
			err = batchOrm.UpdateProofByHash(context.Background(), batch.Hash, batchProof, 100)
			assert.NoError(t, err)
			err = batchOrm.UpdateProvingStatus(context.Background(), batch.Hash, types.ProvingTaskVerified)
			assert.NoError(t, err)
		}

		bundleProof := &message.BundleProof{
			Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Instances: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Vk:        []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		}
		bundles, err := bundleOrm.GetBundles(context.Background(), map[string]interface{}{}, nil, 0)
		assert.NoError(t, err)
		for _, bundle := range bundles {
			err = bundleOrm.UpdateProofAndProvingStatusByHash(context.Background(), bundle.Hash, bundleProof, types.ProvingTaskVerified, 100)
			assert.NoError(t, err)
		}

		assert.Eventually(t, func() bool {
			l2Relayer.ProcessPendingBundles()

			batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, nil, 0)
			assert.NoError(t, err)
			assert.Len(t, batches, 3)
			batches = batches[1:]
			for _, batch := range batches {
				if types.RollupStatus(batch.RollupStatus) != types.RollupFinalized {
					return false
				}

				assert.NotEmpty(t, batch.FinalizeTxHash)
				receipt, getErr := l1Client.TransactionReceipt(context.Background(), common.HexToHash(batch.FinalizeTxHash))
				assert.NoError(t, getErr)
				assert.Equal(t, gethTypes.ReceiptStatusSuccessful, receipt.Status)
			}

			bundles, err := bundleOrm.GetBundles(context.Background(), map[string]interface{}{}, nil, 0)
			assert.NoError(t, err)
			assert.Len(t, bundles, 1)

			bundle := bundles[0]
			if types.RollupStatus(bundle.RollupStatus) != types.RollupFinalized {
				return false
			}
			assert.NotEmpty(t, bundle.FinalizeTxHash)
			receipt, err := l1Client.TransactionReceipt(context.Background(), common.HexToHash(bundle.FinalizeTxHash))
			assert.NoError(t, err)
			assert.Equal(t, gethTypes.ReceiptStatusSuccessful, receipt.Status)
			batches, err = batchOrm.GetBatches(context.Background(), map[string]interface{}{"bundle_hash": bundle.Hash}, nil, 0)
			assert.NoError(t, err)
			assert.Len(t, batches, 2)
			for _, batch := range batches {
				assert.Equal(t, batch.RollupStatus, bundle.RollupStatus)
				assert.Equal(t, bundle.FinalizeTxHash, batch.FinalizeTxHash)
			}

			return true
		}, 30*time.Second, time.Second)

		l2Relayer.StopSenders()
		database.CloseDB(db)
	}
}
