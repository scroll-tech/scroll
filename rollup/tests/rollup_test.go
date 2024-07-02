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

func testCommitBatchAndFinalizeBatch(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	prepareContracts(t)

	// Create L2Relayer
	l2Cfg := rollupApp.Config.L2Config
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, &params.ChainConfig{}, true, relayer.ServiceTypeL2RollupRelayer, nil)
	assert.NoError(t, err)
	defer l2Relayer.StopSenders()

	// Create L1Watcher
	l1Cfg := rollupApp.Config.L1Config
	l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, 0, l1Cfg.Confirmations, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db, nil)

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

	l2BlockOrm := orm.NewL2Block(db)
	err = l2BlockOrm.InsertL2Blocks(context.Background(), blocks)
	assert.NoError(t, err)

	cp := watcher.NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:             100,
		MaxTxNumPerChunk:                10000,
		MaxL1CommitGasPerChunk:          50000000000,
		MaxL1CommitCalldataSizePerChunk: 1000000,
		MaxRowConsumptionPerChunk:       1048319,
		ChunkTimeoutSec:                 300,
	}, &params.ChainConfig{}, db, nil)

	bp := watcher.NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxL1CommitGasPerBatch:          50000000000,
		MaxL1CommitCalldataSizePerBatch: 1000000,
		BatchTimeoutSec:                 300,
	}, &params.ChainConfig{}, db, nil)

	cp.TryProposeChunk()

	batchOrm := orm.NewBatch(db)
	unbatchedChunkIndex, err := batchOrm.GetFirstUnbatchedChunkIndex(context.Background())
	assert.NoError(t, err)

	chunkOrm := orm.NewChunk(db)
	chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), unbatchedChunkIndex, 0)
	assert.NoError(t, err)
	assert.Len(t, chunks, 1)

	bp.TryProposeBatch()

	l2Relayer.ProcessPendingBatches()
	batch, err := batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, batch)

	// fetch rollup events
	assert.Eventually(t, func() bool {
		err = l1Watcher.FetchContractEvent()
		assert.NoError(t, err)
		var statuses []types.RollupStatus
		statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
		return err == nil && len(statuses) == 1 && types.RollupCommitted == statuses[0]
	}, 30*time.Second, time.Second)

	assert.Eventually(t, func() bool {
		batch, err = batchOrm.GetLatestBatch(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, batch)
		assert.NotEmpty(t, batch.CommitTxHash)
		var receipt *gethTypes.Receipt
		receipt, err = l1Client.TransactionReceipt(context.Background(), common.HexToHash(batch.CommitTxHash))
		return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
	}, 30*time.Second, time.Second)

	// add dummy proof
	proof := &message.BatchProof{
		Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		Instances: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		Vk:        []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
	}
	err = batchOrm.UpdateProofByHash(context.Background(), batch.Hash, proof, 100)
	assert.NoError(t, err)
	err = batchOrm.UpdateProvingStatus(context.Background(), batch.Hash, types.ProvingTaskVerified)
	assert.NoError(t, err)

	// process committed batch and check status
	l2Relayer.ProcessCommittedBatches()

	statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupFinalizing, statuses[0])

	// fetch rollup events
	assert.Eventually(t, func() bool {
		err = l1Watcher.FetchContractEvent()
		assert.NoError(t, err)
		var statuses []types.RollupStatus
		statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
		return err == nil && len(statuses) == 1 && types.RollupFinalized == statuses[0]
	}, 30*time.Second, time.Second)

	assert.Eventually(t, func() bool {
		batch, err = batchOrm.GetLatestBatch(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, batch)
		assert.NotEmpty(t, batch.FinalizeTxHash)
		var receipt *gethTypes.Receipt
		receipt, err = l1Client.TransactionReceipt(context.Background(), common.HexToHash(batch.FinalizeTxHash))
		return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
	}, 30*time.Second, time.Second)
}

func testCommitBatchAndFinalizeBatch4844(t *testing.T) {
	compressionTests := []bool{false, true} // false for uncompressed, true for compressed
	for _, compressed := range compressionTests {
		db := setupDB(t)

		prepareContracts(t)

		// Create L2Relayer
		l2Cfg := rollupApp.Config.L2Config
		var chainConfig *params.ChainConfig
		if compressed {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0)}
		} else {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0)}
		}
		l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, chainConfig, true, relayer.ServiceTypeL2RollupRelayer, nil)
		assert.NoError(t, err)

		// Create L1Watcher
		l1Cfg := rollupApp.Config.L1Config
		l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, 0, l1Cfg.Confirmations, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db, nil)

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

		l2BlockOrm := orm.NewL2Block(db)
		err = l2BlockOrm.InsertL2Blocks(context.Background(), blocks)
		assert.NoError(t, err)

		cp := watcher.NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
			MaxBlockNumPerChunk:             100,
			MaxTxNumPerChunk:                10000,
			MaxL1CommitGasPerChunk:          1,
			MaxL1CommitCalldataSizePerChunk: 100000,
			MaxRowConsumptionPerChunk:       1048319,
			ChunkTimeoutSec:                 300,
			MaxUncompressedBatchBytesSize:   math.MaxUint64,
		}, chainConfig, db, nil)

		bp := watcher.NewBatchProposer(context.Background(), &config.BatchProposerConfig{
			MaxL1CommitGasPerBatch:          1,
			MaxL1CommitCalldataSizePerBatch: 100000,
			BatchTimeoutSec:                 300,
			MaxUncompressedBatchBytesSize:   math.MaxUint64,
		}, chainConfig, db, nil)

		cp.TryProposeChunk()

		batchOrm := orm.NewBatch(db)
		unbatchedChunkIndex, err := batchOrm.GetFirstUnbatchedChunkIndex(context.Background())
		assert.NoError(t, err)

		chunkOrm := orm.NewChunk(db)
		chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), unbatchedChunkIndex, 0)
		assert.NoError(t, err)
		assert.Len(t, chunks, 1)

		bp.TryProposeBatch()

		l2Relayer.ProcessPendingBatches()
		batch, err := batchOrm.GetLatestBatch(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, batch)

		// fetch rollup events
		assert.Eventually(t, func() bool {
			err = l1Watcher.FetchContractEvent()
			assert.NoError(t, err)
			var statuses []types.RollupStatus
			statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
			return err == nil && len(statuses) == 1 && types.RollupCommitted == statuses[0]
		}, 30*time.Second, time.Second)

		assert.Eventually(t, func() bool {
			batch, err = batchOrm.GetLatestBatch(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, batch)
			assert.NotEmpty(t, batch.CommitTxHash)
			var receipt *gethTypes.Receipt
			receipt, err = l1Client.TransactionReceipt(context.Background(), common.HexToHash(batch.CommitTxHash))
			return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
		}, 30*time.Second, time.Second)

		// add dummy proof
		proof := &message.BatchProof{
			Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Instances: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Vk:        []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		}
		err = batchOrm.UpdateProofByHash(context.Background(), batch.Hash, proof, 100)
		assert.NoError(t, err)
		err = batchOrm.UpdateProvingStatus(context.Background(), batch.Hash, types.ProvingTaskVerified)
		assert.NoError(t, err)

		// process committed batch and check status
		l2Relayer.ProcessCommittedBatches()

		statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(statuses))
		assert.Equal(t, types.RollupFinalizing, statuses[0])

		// fetch rollup events
		assert.Eventually(t, func() bool {
			err = l1Watcher.FetchContractEvent()
			assert.NoError(t, err)
			var statuses []types.RollupStatus
			statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
			return err == nil && len(statuses) == 1 && types.RollupFinalized == statuses[0]
		}, 30*time.Second, time.Second)

		assert.Eventually(t, func() bool {
			batch, err = batchOrm.GetLatestBatch(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, batch)
			assert.NotEmpty(t, batch.FinalizeTxHash)
			var receipt *gethTypes.Receipt
			receipt, err = l1Client.TransactionReceipt(context.Background(), common.HexToHash(batch.FinalizeTxHash))
			return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
		}, 30*time.Second, time.Second)

		l2Relayer.StopSenders()
		database.CloseDB(db)
	}
}

func testCommitBatchAndFinalizeBatchBeforeAndAfter4844(t *testing.T) {
	compressionTests := []bool{false, true} // false for uncompressed, true for compressed
	for _, compressed := range compressionTests {
		db := setupDB(t)

		prepareContracts(t)

		// Create L2Relayer
		l2Cfg := rollupApp.Config.L2Config
		var chainConfig *params.ChainConfig
		if compressed {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(5), CurieBlock: big.NewInt(5)}
		} else {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(5)}
		}
		l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, chainConfig, true, relayer.ServiceTypeL2RollupRelayer, nil)
		assert.NoError(t, err)

		// Create L1Watcher
		l1Cfg := rollupApp.Config.L1Config
		l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, 0, l1Cfg.Confirmations, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db, nil)

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

		l2BlockOrm := orm.NewL2Block(db)
		err = l2BlockOrm.InsertL2Blocks(context.Background(), blocks)
		assert.NoError(t, err)

		cp := watcher.NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
			MaxBlockNumPerChunk:             100,
			MaxTxNumPerChunk:                10000,
			MaxL1CommitGasPerChunk:          50000000000,
			MaxL1CommitCalldataSizePerChunk: 1000000,
			MaxRowConsumptionPerChunk:       1048319,
			ChunkTimeoutSec:                 300,
			MaxUncompressedBatchBytesSize:   math.MaxUint64,
		}, chainConfig, db, nil)

		bp := watcher.NewBatchProposer(context.Background(), &config.BatchProposerConfig{
			MaxL1CommitGasPerBatch:          50000000000,
			MaxL1CommitCalldataSizePerBatch: 1000000,
			BatchTimeoutSec:                 300,
			MaxUncompressedBatchBytesSize:   math.MaxUint64,
		}, chainConfig, db, nil)

		cp.TryProposeChunk()
		cp.TryProposeChunk()
		bp.TryProposeBatch()
		bp.TryProposeBatch()

		for i := uint64(0); i < 2; i++ {
			l2Relayer.ProcessPendingBatches()
			batchOrm := orm.NewBatch(db)
			batch, err := batchOrm.GetBatchByIndex(context.Background(), i+1)
			assert.NoError(t, err)
			assert.NotNil(t, batch)

			// fetch rollup events
			assert.Eventually(t, func() bool {
				err = l1Watcher.FetchContractEvent()
				assert.NoError(t, err)
				var statuses []types.RollupStatus
				statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
				return err == nil && len(statuses) == 1 && types.RollupCommitted == statuses[0]
			}, 30*time.Second, time.Second)

			assert.Eventually(t, func() bool {
				batch, err = batchOrm.GetBatchByIndex(context.Background(), i+1)
				assert.NoError(t, err)
				assert.NotNil(t, batch)
				assert.NotEmpty(t, batch.CommitTxHash)
				var receipt *gethTypes.Receipt
				receipt, err = l1Client.TransactionReceipt(context.Background(), common.HexToHash(batch.CommitTxHash))
				return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
			}, 30*time.Second, time.Second)

			// add dummy proof
			proof := &message.BatchProof{
				Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
				Instances: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
				Vk:        []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			}
			err = batchOrm.UpdateProofByHash(context.Background(), batch.Hash, proof, 100)
			assert.NoError(t, err)
			err = batchOrm.UpdateProvingStatus(context.Background(), batch.Hash, types.ProvingTaskVerified)
			assert.NoError(t, err)

			// process committed batch and check status
			l2Relayer.ProcessCommittedBatches()

			statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
			assert.NoError(t, err)
			assert.Equal(t, 1, len(statuses))
			assert.Equal(t, types.RollupFinalizing, statuses[0])

			// fetch rollup events
			assert.Eventually(t, func() bool {
				err = l1Watcher.FetchContractEvent()
				assert.NoError(t, err)
				var statuses []types.RollupStatus
				statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
				return err == nil && len(statuses) == 1 && types.RollupFinalized == statuses[0]
			}, 30*time.Second, time.Second)

			assert.Eventually(t, func() bool {
				batch, err = batchOrm.GetBatchByIndex(context.Background(), i+1)
				assert.NoError(t, err)
				assert.NotNil(t, batch)
				assert.NotEmpty(t, batch.FinalizeTxHash)
				var receipt *gethTypes.Receipt
				receipt, err = l1Client.TransactionReceipt(context.Background(), common.HexToHash(batch.FinalizeTxHash))
				return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
			}, 30*time.Second, time.Second)
		}

		l2Relayer.StopSenders()
		database.CloseDB(db)
	}
}

func testCommitBatchAndFinalizeBatchBeforeAndAfterCompression(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	prepareContracts(t)

	// Create L2Relayer
	l2Cfg := rollupApp.Config.L2Config
	chainConfig := &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(5)}
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, chainConfig, true, relayer.ServiceTypeL2RollupRelayer, nil)
	assert.NoError(t, err)
	defer l2Relayer.StopSenders()

	// Create L1Watcher
	l1Cfg := rollupApp.Config.L1Config
	l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, 0, l1Cfg.Confirmations, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db, nil)

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

	l2BlockOrm := orm.NewL2Block(db)
	err = l2BlockOrm.InsertL2Blocks(context.Background(), blocks)
	assert.NoError(t, err)

	cp := watcher.NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:             100,
		MaxTxNumPerChunk:                10000,
		MaxL1CommitGasPerChunk:          50000000000,
		MaxL1CommitCalldataSizePerChunk: 1000000,
		MaxRowConsumptionPerChunk:       1048319,
		ChunkTimeoutSec:                 300,
		MaxUncompressedBatchBytesSize:   math.MaxUint64,
	}, chainConfig, db, nil)

	bp := watcher.NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxL1CommitGasPerBatch:          50000000000,
		MaxL1CommitCalldataSizePerBatch: 1000000,
		BatchTimeoutSec:                 300,
		MaxUncompressedBatchBytesSize:   math.MaxUint64,
	}, chainConfig, db, nil)

	cp.TryProposeChunk()
	cp.TryProposeChunk()
	bp.TryProposeBatch()
	bp.TryProposeBatch()

	for i := uint64(0); i < 2; i++ {
		l2Relayer.ProcessPendingBatches()
		batchOrm := orm.NewBatch(db)
		batch, err := batchOrm.GetBatchByIndex(context.Background(), i+1)
		assert.NoError(t, err)
		assert.NotNil(t, batch)

		// fetch rollup events
		assert.Eventually(t, func() bool {
			err = l1Watcher.FetchContractEvent()
			assert.NoError(t, err)
			var statuses []types.RollupStatus
			statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
			return err == nil && len(statuses) == 1 && types.RollupCommitted == statuses[0]
		}, 30*time.Second, time.Second)

		assert.Eventually(t, func() bool {
			batch, err = batchOrm.GetBatchByIndex(context.Background(), i+1)
			assert.NoError(t, err)
			assert.NotNil(t, batch)
			assert.NotEmpty(t, batch.CommitTxHash)
			var receipt *gethTypes.Receipt
			receipt, err = l1Client.TransactionReceipt(context.Background(), common.HexToHash(batch.CommitTxHash))
			return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
		}, 30*time.Second, time.Second)

		// add dummy proof
		proof := &message.BatchProof{
			Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Instances: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			Vk:        []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		}
		err = batchOrm.UpdateProofByHash(context.Background(), batch.Hash, proof, 100)
		assert.NoError(t, err)
		err = batchOrm.UpdateProvingStatus(context.Background(), batch.Hash, types.ProvingTaskVerified)
		assert.NoError(t, err)

		// process committed batch and check status
		l2Relayer.ProcessCommittedBatches()

		statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(statuses))
		assert.Equal(t, types.RollupFinalizing, statuses[0])

		// fetch rollup events
		assert.Eventually(t, func() bool {
			err = l1Watcher.FetchContractEvent()
			assert.NoError(t, err)
			var statuses []types.RollupStatus
			statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
			return err == nil && len(statuses) == 1 && types.RollupFinalized == statuses[0]
		}, 30*time.Second, time.Second)

		assert.Eventually(t, func() bool {
			batch, err = batchOrm.GetBatchByIndex(context.Background(), i+1)
			assert.NoError(t, err)
			assert.NotNil(t, batch)
			assert.NotEmpty(t, batch.FinalizeTxHash)
			var receipt *gethTypes.Receipt
			receipt, err = l1Client.TransactionReceipt(context.Background(), common.HexToHash(batch.FinalizeTxHash))
			return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
		}, 30*time.Second, time.Second)
	}
}
