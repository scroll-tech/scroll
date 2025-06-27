package tests

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/types"

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
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, &params.ChainConfig{}, relayer.ServiceTypeL2RollupRelayer, nil)
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

func testCommitBatchAndFinalizeBundleCodecV7(t *testing.T) {
	db := setupDB(t)

	prepareContracts(t)

	chainConfig := &params.ChainConfig{
		LondonBlock:    big.NewInt(0),
		BernoulliBlock: big.NewInt(0),
		CurieBlock:     big.NewInt(0),
		DarwinTime:     new(uint64),
		DarwinV2Time:   new(uint64),
		EuclidTime:     new(uint64),
		EuclidV2Time:   new(uint64),
	}

	// Create L2Relayer
	l2Cfg := rollupApp.Config.L2Config
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, chainConfig, relayer.ServiceTypeL2RollupRelayer, nil)
	require.NoError(t, err)

	defer l2Relayer.StopSenders()
	defer database.CloseDB(db)

	// add some blocks to db
	var blocks []*encoding.Block
	genesis, err := l2Client.HeaderByNumber(context.Background(), big.NewInt(0))
	require.NoError(t, err)

	var l1MessageIndex uint64 = 0
	parentHash := genesis.Hash()
	for i := int64(0); i < 10; i++ {
		header := gethTypes.Header{
			Number:     big.NewInt(i + 1),
			ParentHash: parentHash,
			Difficulty: big.NewInt(i + 1),
			BaseFee:    big.NewInt(i + 1),
			Root:       common.HexToHash("0x1"),
		}
		fmt.Println("block number: ", i+1, header.Hash(), parentHash)
		var transactions []*gethTypes.TransactionData
		if i%2 == 0 {
			txs := []*gethTypes.Transaction{
				gethTypes.NewTx(&gethTypes.L1MessageTx{
					QueueIndex: l1MessageIndex,
					Gas:        0,
					To:         &common.Address{1, 2, 3},
					Value:      big.NewInt(10),
					Data:       nil,
					Sender:     common.Address{1, 2, 3},
				}),
			}
			transactions = append(transactions, encoding.TxsToTxsData(txs)...)
			l1MessageIndex++
		}

		blocks = append(blocks, &encoding.Block{
			Header:       &header,
			Transactions: transactions,
			WithdrawRoot: common.HexToHash("0x2"),
		})
		parentHash = header.Hash()
	}

	cp := watcher.NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:           100,
		MaxL2GasPerChunk:              math.MaxUint64,
		ChunkTimeoutSec:               300,
		MaxUncompressedBatchBytesSize: math.MaxUint64,
	}, encoding.CodecV7, chainConfig, db, nil)

	bap := watcher.NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxChunksPerBatch:             math.MaxInt32,
		BatchTimeoutSec:               300,
		MaxUncompressedBatchBytesSize: math.MaxUint64,
	}, encoding.CodecV7, chainConfig, db, nil)

	bup := watcher.NewBundleProposer(context.Background(), &config.BundleProposerConfig{
		MaxBatchNumPerBundle: 2,
		BundleTimeoutSec:     300,
	}, encoding.CodecV7, chainConfig, db, nil)

	l2BlockOrm := orm.NewL2Block(db)
	batchOrm := orm.NewBatch(db)
	bundleOrm := orm.NewBundle(db)

	var batch1ExpectedLastL1MessageQueueHash common.Hash
	{
		fmt.Println("insert first 5 blocks ------------------------")
		err = l2BlockOrm.InsertL2Blocks(context.Background(), blocks[:5])
		require.NoError(t, err)
		batch1ExpectedLastL1MessageQueueHash, err = encoding.MessageQueueV2ApplyL1MessagesFromBlocks(common.Hash{}, blocks[:5])
		require.NoError(t, err)

		cp.TryProposeChunk()
		bap.TryProposeBatch()
	}

	var batch2ExpectedLastL1MessageQueueHash common.Hash
	{
		fmt.Println("insert next 3 blocks ------------------------")
		err = l2BlockOrm.InsertL2Blocks(context.Background(), blocks[5:8])
		for _, block := range blocks[5:8] {
			fmt.Println("insert[5:8] block number: ", block.Header.Number, block.Header.Hash())
		}
		require.NoError(t, err)
		batch2ExpectedLastL1MessageQueueHash, err = encoding.MessageQueueV2ApplyL1MessagesFromBlocks(batch1ExpectedLastL1MessageQueueHash, blocks[5:8])
		require.NoError(t, err)

		cp.TryProposeChunk()
		bap.TryProposeBatch()
	}

	var batch3ExpectedLastL1MessageQueueHash common.Hash
	{
		fmt.Println("insert last 2 blocks ------------------------")
		err = l2BlockOrm.InsertL2Blocks(context.Background(), blocks[8:])
		for _, block := range blocks[8:] {
			fmt.Println("insert[:8] block number: ", block.Header.Number, block.Header.Hash())
		}
		require.NoError(t, err)
		batch3ExpectedLastL1MessageQueueHash, err = encoding.MessageQueueV2ApplyL1MessagesFromBlocks(batch2ExpectedLastL1MessageQueueHash, blocks[8:])
		require.NoError(t, err)

		cp.TryProposeChunk()
		bap.TryProposeBatch()
	}

	var batches []*orm.Batch
	// make sure that batches are created as expected
	require.Eventually(t, func() bool {
		batches, err = batchOrm.GetBatches(context.Background(), map[string]interface{}{}, nil, 0)
		if err != nil {
			return false
		}
		if len(batches) != 4 {
			return false
		}

		// batches[0] is the genesis batch, no need to check

		// assert correctness of L1 message queue hashes
		require.Equal(t, common.Hash{}, common.HexToHash(batches[1].PrevL1MessageQueueHash))
		require.Equal(t, batch1ExpectedLastL1MessageQueueHash, common.HexToHash(batches[1].PostL1MessageQueueHash))
		require.Equal(t, batch1ExpectedLastL1MessageQueueHash, common.HexToHash(batches[2].PrevL1MessageQueueHash))
		require.Equal(t, batch2ExpectedLastL1MessageQueueHash, common.HexToHash(batches[2].PostL1MessageQueueHash))
		require.Equal(t, batch2ExpectedLastL1MessageQueueHash, common.HexToHash(batches[3].PrevL1MessageQueueHash))
		require.Equal(t, batch3ExpectedLastL1MessageQueueHash, common.HexToHash(batches[3].PostL1MessageQueueHash))

		return true
	}, 30*time.Second, time.Second)

	// Nothing should happen since no batch is committed yet.
	{
		bup.TryProposeBundle()
		bundles, err := bundleOrm.GetBundles(context.Background(), map[string]interface{}{}, nil, 0)
		require.NoError(t, err)
		require.Len(t, bundles, 0)
	}

	// simulate batches 2 and 3 being submitted together in a single transaction
	err = db.Transaction(func(dbTX *gorm.DB) error {
		if err = batchOrm.UpdateCommitTxHashAndRollupStatus(context.Background(), batches[1].Hash, "0xdefdef", types.RollupCommitted, dbTX); err != nil {
			return fmt.Errorf("UpdateCommitTxHashAndRollupStatus failed for batch %d: %s, err %v", batches[1].Index, batches[1].Hash, err)
		}

		for _, batch := range batches[2:] {
			if err = batchOrm.UpdateCommitTxHashAndRollupStatus(context.Background(), batch.Hash, "0xabcabc", types.RollupCommitted, dbTX); err != nil {
				return fmt.Errorf("UpdateCommitTxHashAndRollupStatus failed for batch %d: %s, err %v", batch.Index, batch.Hash, err)
			}
		}

		return nil
	})
	require.NoError(t, err)

	// We only allow bundles up to 2 batches. We should have 2 bundles:
	//  1. batch 1 -> because it was committed by itself and the next set of batches could not fit the bundle
	//  2. batch 2 and 3 -> because they were committed together in a single transaction
	{
		// need to propose 2 times to get 2 bundles with all batches
		bup.TryProposeBundle()
		bup.TryProposeBundle()

		bundles, err := bundleOrm.GetBundles(context.Background(), map[string]interface{}{}, nil, 0)
		require.NoError(t, err)
		require.Len(t, bundles, 2)

		require.Equal(t, bundles[0].StartBatchIndex, batches[1].Index)
		require.Equal(t, bundles[0].EndBatchIndex, batches[1].Index)
		require.Equal(t, bundles[0].StartBatchHash, batches[1].Hash)
		require.Equal(t, bundles[0].EndBatchHash, batches[1].Hash)

		require.Equal(t, bundles[1].StartBatchIndex, batches[2].Index)
		require.Equal(t, bundles[1].EndBatchIndex, batches[3].Index)
		require.Equal(t, bundles[1].StartBatchHash, batches[2].Hash)
		require.Equal(t, bundles[1].EndBatchHash, batches[3].Hash)
	}

	// TODO: update mock bridge contract ABI to support new methods
	//  - simulate proof generation -> all batches and bundle are verified
	//  - make sure batches are actually committed and bundles are finalized

	// simulate proof generation -> all batches and bundle are verified
	//{
	//	batchProof := &message.OpenVMBatchProof{}
	//	batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, nil, 0)
	//	require.NoError(t, err)
	//	batches = batches[1:]
	//	for _, batch := range batches {
	//		err = batchOrm.UpdateProofByHash(context.Background(), batch.Hash, batchProof, 100)
	//		require.NoError(t, err)
	//		err = batchOrm.UpdateProvingStatus(context.Background(), batch.Hash, types.ProvingTaskVerified)
	//		require.NoError(t, err)
	//	}
	//
	//	bundleProof := &message.OpenVMBundleProof{}
	//	bundles, err := bundleOrm.GetBundles(context.Background(), map[string]interface{}{}, nil, 0)
	//	require.NoError(t, err)
	//	for _, bundle := range bundles {
	//		err = bundleOrm.UpdateProofAndProvingStatusByHash(context.Background(), bundle.Hash, bundleProof, types.ProvingTaskVerified, 100)
	//		require.NoError(t, err)
	//	}
	//}
}
