package watcher

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	"scroll-tech/common/types"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
	"scroll-tech/rollup/internal/utils"
)

func testBundleProposerLimits(t *testing.T) {
	tests := []struct {
		name                         string
		maxBatchNumPerBundle         uint64
		bundleTimeoutSec             uint64
		expectedBundlesLen           int
		expectedBatchesInFirstBundle uint64 // only be checked when expectedBundlesLen > 0
	}{
		{
			name:                 "NoLimitReached",
			maxBatchNumPerBundle: math.MaxUint64,
			bundleTimeoutSec:     math.MaxUint32,
			expectedBundlesLen:   0,
		},
		{
			name:                         "Timeout",
			maxBatchNumPerBundle:         math.MaxUint64,
			bundleTimeoutSec:             0,
			expectedBundlesLen:           1,
			expectedBatchesInFirstBundle: 2,
		},
		{
			name:                 "maxBatchNumPerBundleIs0",
			maxBatchNumPerBundle: 0,
			bundleTimeoutSec:     math.MaxUint32,
			expectedBundlesLen:   0,
		},
		{
			name:                         "maxBatchNumPerBundleIs1",
			maxBatchNumPerBundle:         1,
			bundleTimeoutSec:             math.MaxUint32,
			expectedBundlesLen:           1,
			expectedBatchesInFirstBundle: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			defer database.CloseDB(db)

			// Add genesis batch.
			block := &encoding.Block{
				Header: &gethTypes.Header{
					Number: big.NewInt(0),
				},
				RowConsumption: &gethTypes.RowConsumption{},
			}
			chunk := &encoding.Chunk{
				Blocks: []*encoding.Block{block},
			}
			chunkOrm := orm.NewChunk(db)
			_, err := chunkOrm.InsertChunk(context.Background(), chunk, utils.CodecConfig{Version: encoding.CodecV0}, utils.ChunkMetrics{})
			assert.NoError(t, err)
			batch := &encoding.Batch{
				Index:                      0,
				TotalL1MessagePoppedBefore: 0,
				ParentBatchHash:            common.Hash{},
				Chunks:                     []*encoding.Chunk{chunk},
			}
			batchOrm := orm.NewBatch(db)
			_, err = batchOrm.InsertBatch(context.Background(), batch, utils.CodecConfig{Version: encoding.CodecV0}, utils.BatchMetrics{})
			assert.NoError(t, err)

			l2BlockOrm := orm.NewL2Block(db)
			err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
			assert.NoError(t, err)

			chainConfig := &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64)}

			cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
				MaxBlockNumPerChunk:             1,
				MaxTxNumPerChunk:                math.MaxUint64,
				MaxL1CommitGasPerChunk:          math.MaxUint64,
				MaxL1CommitCalldataSizePerChunk: math.MaxUint64,
				MaxRowConsumptionPerChunk:       math.MaxUint64,
				ChunkTimeoutSec:                 math.MaxUint32,
				GasCostIncreaseMultiplier:       1,
				MaxUncompressedBatchBytesSize:   math.MaxUint64,
			}, chainConfig, db, nil)

			bap := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
				MaxL1CommitGasPerBatch:          math.MaxUint64,
				MaxL1CommitCalldataSizePerBatch: math.MaxUint64,
				BatchTimeoutSec:                 0,
				GasCostIncreaseMultiplier:       1,
				MaxUncompressedBatchBytesSize:   math.MaxUint64,
			}, chainConfig, db, nil)

			cp.TryProposeChunk()  // chunk1 contains block1
			bap.TryProposeBatch() // batch1 contains chunk1
			cp.TryProposeChunk()  // chunk2 contains block2
			bap.TryProposeBatch() // batch2 contains chunk2

			bup := NewBundleProposer(context.Background(), &config.BundleProposerConfig{
				MaxBatchNumPerBundle: tt.maxBatchNumPerBundle,
				BundleTimeoutSec:     tt.bundleTimeoutSec,
			}, chainConfig, db, nil)

			bup.TryProposeBundle()

			bundleOrm := orm.NewBundle(db)
			bundles, err := bundleOrm.GetBundles(context.Background(), map[string]interface{}{}, []string{}, 0)
			assert.NoError(t, err)
			assert.Len(t, bundles, tt.expectedBundlesLen)
			if tt.expectedBundlesLen > 0 {
				assert.Equal(t, uint64(1), bundles[0].StartBatchIndex)
				assert.Equal(t, tt.expectedBatchesInFirstBundle, bundles[0].EndBatchIndex)
				assert.Equal(t, types.RollupPending, types.RollupStatus(bundles[0].RollupStatus))
				assert.Equal(t, types.ProvingTaskUnassigned, types.ProvingStatus(bundles[0].ProvingStatus))
			}
		})
	}
}

func testBundleProposerRespectHardforks(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	chainConfig := &params.ChainConfig{
		BernoulliBlock: big.NewInt(1),
		CurieBlock:     big.NewInt(2),
		DarwinTime:     func() *uint64 { t := uint64(4); return &t }(),
	}

	// Add genesis batch.
	block := &encoding.Block{
		Header: &gethTypes.Header{
			Number: big.NewInt(0),
		},
		RowConsumption: &gethTypes.RowConsumption{},
	}
	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{block},
	}
	chunkOrm := orm.NewChunk(db)
	_, err := chunkOrm.InsertChunk(context.Background(), chunk, utils.CodecConfig{Version: encoding.CodecV0}, utils.ChunkMetrics{})
	assert.NoError(t, err)
	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
	}
	batchOrm := orm.NewBatch(db)
	_, err = batchOrm.InsertBatch(context.Background(), batch, utils.CodecConfig{Version: encoding.CodecV0}, utils.BatchMetrics{})
	assert.NoError(t, err)

	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:             math.MaxUint64,
		MaxTxNumPerChunk:                math.MaxUint64,
		MaxL1CommitGasPerChunk:          math.MaxUint64,
		MaxL1CommitCalldataSizePerChunk: math.MaxUint64,
		MaxRowConsumptionPerChunk:       math.MaxUint64,
		ChunkTimeoutSec:                 0,
		GasCostIncreaseMultiplier:       1,
		MaxUncompressedBatchBytesSize:   math.MaxUint64,
	}, chainConfig, db, nil)

	block = readBlockFromJSON(t, "../../../testdata/blockTrace_02.json")
	for i := int64(1); i <= 60; i++ {
		block.Header.Number = big.NewInt(i)
		block.Header.Time = uint64(i)
		err = orm.NewL2Block(db).InsertL2Blocks(context.Background(), []*encoding.Block{block})
		assert.NoError(t, err)
	}

	for i := 0; i < 5; i++ {
		cp.TryProposeChunk()
	}

	bap := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxL1CommitGasPerBatch:          math.MaxUint64,
		MaxL1CommitCalldataSizePerBatch: math.MaxUint64,
		BatchTimeoutSec:                 0,
		GasCostIncreaseMultiplier:       1,
		MaxUncompressedBatchBytesSize:   math.MaxUint64,
	}, chainConfig, db, nil)

	for i := 0; i < 5; i++ {
		bap.TryProposeBatch()
	}

	bup := NewBundleProposer(context.Background(), &config.BundleProposerConfig{
		MaxBatchNumPerBundle: math.MaxUint64,
		BundleTimeoutSec:     0,
	}, chainConfig, db, nil)

	for i := 0; i < 5; i++ {
		bup.TryProposeBundle()
	}

	bundleOrm := orm.NewBundle(db)
	bundles, err := bundleOrm.GetBundles(context.Background(), map[string]interface{}{}, []string{}, 0)
	assert.NoError(t, err)
	assert.Len(t, bundles, 1)

	expectedStartBatchIndices := []uint64{3}
	expectedEndChunkIndices := []uint64{3}
	for i, bundle := range bundles {
		assert.Equal(t, expectedStartBatchIndices[i], bundle.StartBatchIndex)
		assert.Equal(t, expectedEndChunkIndices[i], bundle.EndBatchIndex)
	}
}
