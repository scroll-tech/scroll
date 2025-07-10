package relayer

import (
	"context"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/watcher"
	"scroll-tech/rollup/internal/orm"
	butils "scroll-tech/rollup/internal/utils"
)

type FullRecovery struct {
	ctx       context.Context
	cfg       *config.Config
	genesis   *core.Genesis
	db        *gorm.DB
	blockORM  *orm.L2Block
	chunkORM  *orm.Chunk
	batchORM  *orm.Batch
	bundleORM *orm.Bundle

	chunkProposer    *watcher.ChunkProposer
	batchProposer    *watcher.BatchProposer
	bundleProposer   *watcher.BundleProposer
	l2Watcher        *watcher.L2WatcherClient
	l1Client         *ethclient.Client
	l1Reader         *l1.Reader
	beaconNodeClient *blob_client.BeaconNodeClient
}

func NewFullRecovery(ctx context.Context, cfg *config.Config, genesis *core.Genesis, db *gorm.DB, chunkProposer *watcher.ChunkProposer, batchProposer *watcher.BatchProposer, bundleProposer *watcher.BundleProposer, l2Watcher *watcher.L2WatcherClient, l1Client *ethclient.Client, l1Reader *l1.Reader) (*FullRecovery, error) {
	beaconNodeClient, err := blob_client.NewBeaconNodeClient(cfg.L1Config.BeaconNodeEndpoint)
	if err != nil {
		return nil, fmt.Errorf("create blob client failed: %v", err)
	}

	return &FullRecovery{
		ctx:       ctx,
		cfg:       cfg,
		genesis:   genesis,
		db:        db,
		blockORM:  orm.NewL2Block(db),
		chunkORM:  orm.NewChunk(db),
		batchORM:  orm.NewBatch(db),
		bundleORM: orm.NewBundle(db),

		chunkProposer:    chunkProposer,
		batchProposer:    batchProposer,
		bundleProposer:   bundleProposer,
		l2Watcher:        l2Watcher,
		l1Client:         l1Client,
		l1Reader:         l1Reader,
		beaconNodeClient: beaconNodeClient,
	}, nil
}

// RestoreFullPreviousState restores the full state from L1.
// The DB state should be clean: the latest batch in the DB should be finalized on L1. This function will
// restore all batches between the latest finalized batch in the DB and the latest finalized batch on L1.
func (f *FullRecovery) RestoreFullPreviousState() error {
	log.Info("Restoring full previous state")

	// 1. Get latest finalized batch stored in DB
	latestDBBatch, err := f.batchORM.GetLatestBatch(f.ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest batch from DB: %w", err)
	}

	log.Info("Latest finalized batch in DB", "batch", latestDBBatch.Index, "hash", latestDBBatch.Hash)

	// 2. Get latest finalized L1 block
	latestFinalizedL1Block, err := f.l1Reader.GetLatestFinalizedBlockNumber()
	if err != nil {
		return fmt.Errorf("failed to get latest finalized L1 block number: %w", err)
	}

	log.Info("Latest finalized L1 block number", "latest finalized L1 block", latestFinalizedL1Block)

	// 3. Get latest finalized batch from contract (at latest finalized L1 block)
	latestFinalizedBatchContract, err := f.l1Reader.LatestFinalizedBatchIndex(latestFinalizedL1Block)
	if err != nil {
		return fmt.Errorf("failed to get latest finalized batch: %w", err)
	}

	log.Info("Latest finalized batch from L1 contract", "latest finalized batch", latestFinalizedBatchContract, "at latest finalized L1 block", latestFinalizedL1Block)

	// 4. Get batches one by one from stored in DB to latest finalized batch.
	var fromBlock uint64
	if latestDBBatch.Index > 0 {
		receipt, err := f.l1Client.TransactionReceipt(f.ctx, common.HexToHash(latestDBBatch.CommitTxHash))
		if err != nil {
			return fmt.Errorf("failed to get transaction receipt of latest DB batch finalization transaction: %w", err)
		}
		fromBlock = receipt.BlockNumber.Uint64()
	} else {
		fromBlock = f.cfg.L1Config.StartHeight
	}

	log.Info("Fetching rollup events from L1", "from block", fromBlock, "to block", latestFinalizedL1Block, "from batch", latestDBBatch.Index, "to batch", latestFinalizedBatchContract)

	commitsHeapMap := common.NewHeapMap[uint64, *l1.CommitBatchEvent](func(event *l1.CommitBatchEvent) uint64 {
		return event.BatchIndex().Uint64()
	})
	batchEventsHeap := common.NewHeap[*batchEvents]()
	var bundles [][]*batchEvents

	err = f.l1Reader.FetchRollupEventsInRangeWithCallback(fromBlock, latestFinalizedL1Block, func(event l1.RollupEvent) bool {
		// We're only interested in batches that are newer than the latest finalized batch in the DB.
		if event.BatchIndex().Uint64() <= latestDBBatch.Index {
			return true
		}

		switch event.Type() {
		case l1.CommitEventType:
			commitEvent := event.(*l1.CommitBatchEvent)
			commitsHeapMap.Push(commitEvent)

		case l1.FinalizeEventType:
			finalizeEvent := event.(*l1.FinalizeBatchEvent)

			var bundle []*batchEvents

			// with bundles all committed batches until this finalized batch are finalized in the same bundle
			for commitsHeapMap.Len() > 0 {
				commitEvent := commitsHeapMap.Peek()
				if commitEvent.BatchIndex().Uint64() > finalizeEvent.BatchIndex().Uint64() {
					break
				}

				bEvents := newBatchEvents(commitEvent, finalizeEvent)
				commitsHeapMap.Pop()
				batchEventsHeap.Push(bEvents)
				bundle = append(bundle, bEvents)
			}

			bundles = append(bundles, bundle)

			// Stop fetching rollup events if we reached the latest finalized batch.
			if finalizeEvent.BatchIndex().Uint64() >= latestFinalizedBatchContract {
				return false
			}
		case l1.RevertEventV0Type:
			// We ignore reverted batches.
			commitsHeapMap.RemoveByKey(event.BatchIndex().Uint64())
		case l1.RevertEventV7Type:
			// We ignore reverted batches.

			revertBatch, ok := event.(*l1.RevertBatchEventV7)
			if !ok {
				log.Error(fmt.Sprintf("unexpected type of revert event: %T, expected RevertEventV7Type", event))
				return false
			}

			// delete all batches from revertBatch.StartBatchIndex (inclusive) to revertBatch.FinishBatchIndex (inclusive)
			for i := revertBatch.StartBatchIndex().Uint64(); i <= revertBatch.FinishBatchIndex().Uint64(); i++ {
				commitsHeapMap.RemoveByKey(i)
			}
		}

		return true
	})
	if err != nil {
		return fmt.Errorf("failed to fetch rollup events: %w", err)
	}

	// 5. Process all finalized batches: fetch L2 blocks and reproduce chunks and batches.
	var batches []*batchEvents
	for batchEventsHeap.Len() > 0 {
		nextBatch := batchEventsHeap.Pop().Value()
		batches = append(batches, nextBatch)
	}

	if err = f.processFinalizedBatches(batches); err != nil {
		return fmt.Errorf("failed to process finalized batches: %w", err)
	}

	// 6. Create bundles if needed.
	for _, bundle := range bundles {
		var dbBatches []*orm.Batch
		var lastBatchInBundle *orm.Batch

		for _, batch := range bundle {
			dbBatch, err := f.batchORM.GetBatchByIndex(f.ctx, batch.commit.BatchIndex().Uint64())
			if err != nil {
				return fmt.Errorf("failed to get batch by index for bundle generation: %w", err)
			}
			// Bundles are only supported for codec version 3 and above.
			if encoding.CodecVersion(dbBatch.CodecVersion) < encoding.CodecV3 {
				break
			}

			dbBatches = append(dbBatches, dbBatch)
			lastBatchInBundle = dbBatch
		}

		if len(dbBatches) == 0 {
			continue
		}

		err = f.db.Transaction(func(dbTX *gorm.DB) error {
			newBundle, err := f.bundleORM.InsertBundle(f.ctx, dbBatches, encoding.CodecVersion(lastBatchInBundle.CodecVersion), dbTX)
			if err != nil {
				return fmt.Errorf("failed to insert bundle to DB: %w", err)
			}
			if err = f.batchORM.UpdateBundleHashInRange(f.ctx, newBundle.StartBatchIndex, newBundle.EndBatchIndex, newBundle.Hash, dbTX); err != nil {
				return fmt.Errorf("failed to update bundle_hash %s for batches (%d to %d): %w", newBundle.Hash, newBundle.StartBatchIndex, newBundle.EndBatchIndex, err)
			}

			if err = f.bundleORM.UpdateFinalizeTxHashAndRollupStatus(f.ctx, newBundle.Hash, lastBatchInBundle.FinalizeTxHash, types.RollupFinalized, dbTX); err != nil {
				return fmt.Errorf("failed to update finalize tx hash and rollup status for bundle %s: %w", newBundle.Hash, err)
			}

			if err = f.bundleORM.UpdateProvingStatus(f.ctx, newBundle.Hash, types.ProvingTaskVerified, dbTX); err != nil {
				return fmt.Errorf("failed to update proving status for bundle %s: %w", newBundle.Hash, err)
			}

			log.Info("Inserted bundle", "hash", newBundle.Hash, "start batch index", newBundle.StartBatchIndex, "end batch index", newBundle.EndBatchIndex)

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to insert bundle in DB transaction: %w", err)
		}
	}

	return nil
}

func (f *FullRecovery) processFinalizedBatches(batches []*batchEvents) error {
	if len(batches) == 0 {
		return fmt.Errorf("no finalized batches to process")
	}

	firstBatch := batches[0]
	lastBatch := batches[len(batches)-1]

	log.Info("Processing finalized batches", "first batch", firstBatch.commit.BatchIndex(), "hash", firstBatch.commit.BatchHash(), "last batch", lastBatch.commit.BatchIndex(), "hash", lastBatch.commit.BatchHash())

	// Since multiple CommitBatch events per transaction is introduced >= CodecV7,
	// with one transaction carrying multiple blobs,
	// each CommitBatch event corresponds to a blob containing block range data.
	// To correctly process these events, we need to:
	// 1. Parsing the associated blob data to extract the block range for each event
	// 2. Tracking the parent batch hash for each processed CommitBatch event, to:
	//   - Validate the batch hash, since parent batch hash is needed to calculate the batch hash
	//   - Derive the index of the current batch by the number of parent batch hashes tracked
	// In commitBatches and commitAndFinalizeBatch, the parent batch hash is passed in calldata,
	// so that we can use it to get the first batch's parent batch hash, and derive the rest.
	// The index map serves this purpose with:
	// Key:   commit transaction hash
	// Value: parent batch hashes (in order) for each processed CommitBatch event in the transaction
	txBlobIndexMap := make(map[common.Hash][]common.Hash)
	for _, b := range batches {
		args, err := f.l1Reader.FetchCommitTxData(b.commit)
		if err != nil {
			return fmt.Errorf("failed to fetch commit tx data of batch %d, tx hash: %v, err: %w", firstBatch.commit.BatchIndex().Uint64(), firstBatch.commit.TxHash().Hex(), err)
		}

		// all batches we process here will be > CodecV7 since that is the minimum codec version for permissionless batches
		if args.Version < 7 {
			return fmt.Errorf("unsupported codec version: %v, batch index: %v, tx hash: %s", args.Version, firstBatch.commit.BatchIndex().Uint64(), firstBatch.commit.TxHash().Hex())
		}

		codec, err := encoding.CodecFromVersion(encoding.CodecVersion(args.Version))
		if err != nil {
			return fmt.Errorf("unsupported codec version: %v, err: %w", args.Version, err)
		}

		// we append the batch hash to the slice for the current commit transaction after processing the batch.
		// that means the current index of the batch within the transaction is len(txBlobIndexMap[vlog.TxHash]).
		currentIndex := len(txBlobIndexMap[b.commit.TxHash()])
		if currentIndex >= len(args.BlobHashes) {
			return fmt.Errorf("commit transaction %s has %d blobs, but trying to access index %d (batch index %d)",
				b.commit.TxHash(), len(args.BlobHashes), currentIndex, b.commit.BatchIndex().Uint64())
		}
		blobVersionedHash := args.BlobHashes[currentIndex]

		// validate the batch hash
		var parentBatchHash common.Hash
		if currentIndex == 0 {
			parentBatchHash = args.ParentBatchHash
		} else {
			// here we need to subtract 1 from the current index to get the parent batch hash.
			parentBatchHash = txBlobIndexMap[b.commit.TxHash()][currentIndex-1]
		}
		calculatedBatch, err := codec.NewDABatchFromParams(b.commit.BatchIndex().Uint64(), blobVersionedHash, parentBatchHash)
		if err != nil {
			return fmt.Errorf("failed to create new DA batch from params, batch index: %d, err: %w", b.commit.BatchIndex().Uint64(), err)
		}
		if calculatedBatch.Hash() != b.commit.BatchHash() {
			return fmt.Errorf("batch hash mismatch for batch %d, expected: %s, got: %s", b.commit.BatchIndex(), b.commit.BatchHash().String(), calculatedBatch.Hash().String())
		}

		txBlobIndexMap[b.commit.TxHash()] = append(txBlobIndexMap[b.commit.TxHash()], b.commit.BatchHash())

		if err = f.insertBatchIntoDB(b, codec, blobVersionedHash); err != nil {
			return fmt.Errorf("failed to insert batch into DB, batch index: %d, err: %w", b.commit.BatchIndex().Uint64(), err)
		}

		log.Info("Processed batch", "index", b.commit.BatchIndex(), "hash", b.commit.BatchHash(), "commit tx hash", b.commit.TxHash().Hex(), "finalize tx hash", b.finalize.TxHash().Hex(), "blob versioned hash", blobVersionedHash.String(), "parent batch hash", parentBatchHash.String())
	}

	return nil
}

func (f *FullRecovery) insertBatchIntoDB(batch *batchEvents, codec encoding.Codec, blobVersionedHash common.Hash) error {
	// 5.1 Fetch block time.
	blockHeader, err := f.l1Reader.FetchBlockHeaderByNumber(batch.commit.BlockNumber())
	if err != nil {
		return fmt.Errorf("failed to fetch block header by number %d: %w", batch.commit.BlockNumber(), err)
	}

	// 5.2 Fetch blob data for batch.
	daBlocks, err := f.getBatchBlockRangeFromBlob(codec, blobVersionedHash, blockHeader.Time)
	if err != nil {
		return fmt.Errorf("failed to get batch block range from blob %s: %w", blobVersionedHash.Hex(), err)
	}
	lastBlock := daBlocks[len(daBlocks)-1]

	// 5.2. Fetch L2 blocks for the entire batch.
	if err = f.l2Watcher.TryFetchRunningMissingBlocks(lastBlock.Number()); err != nil {
		return fmt.Errorf("failed to fetch L2 blocks: %w", err)
	}

	// 5.3. Reproduce chunk. Since we don't know the internals of a batch we just create 1 chunk per batch.
	start := daBlocks[0].Number()
	end := lastBlock.Number()

	// get last chunk from DB
	lastChunk, err := f.chunkORM.GetLatestChunk(f.ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest chunk from DB: %w", err)
	}

	blocks, err := f.blockORM.GetL2BlocksInRange(f.ctx, start, end)
	if err != nil {
		return fmt.Errorf("failed to get L2 blocks in range: %w", err)
	}

	log.Info("Reproducing chunk", "start block", start, "end block", end)

	var chunk encoding.Chunk
	chunk.Blocks = blocks
	chunk.PrevL1MessageQueueHash = common.HexToHash(lastChunk.PostL1MessageQueueHash)
	chunk.PostL1MessageQueueHash, err = encoding.MessageQueueV2ApplyL1MessagesFromBlocks(chunk.PrevL1MessageQueueHash, blocks)
	if err != nil {
		return fmt.Errorf("failed to apply L1 messages from blocks: %w", err)
	}

	metrics, err := butils.CalculateChunkMetrics(&chunk, codec.Version())
	if err != nil {
		return fmt.Errorf("failed to calculate chunk metrics: %w", err)
	}

	var dbChunk *orm.Chunk
	err = f.db.Transaction(func(dbTX *gorm.DB) error {
		dbChunk, err = f.chunkORM.InsertChunk(f.ctx, &chunk, codec.Version(), *metrics, dbTX)
		if err != nil {
			return fmt.Errorf("failed to insert chunk to DB: %w", err)
		}
		if err := f.blockORM.UpdateChunkHashInRange(f.ctx, dbChunk.StartBlockNumber, dbChunk.EndBlockNumber, dbChunk.Hash, dbTX); err != nil {
			return fmt.Errorf("failed to update chunk_hash for l2_blocks (chunk hash: %s, start block: %d, end block: %d): %w", dbChunk.Hash, dbChunk.StartBlockNumber, dbChunk.EndBlockNumber, err)
		}

		if err = f.chunkORM.UpdateProvingStatus(f.ctx, dbChunk.Hash, types.ProvingTaskVerified, dbTX); err != nil {
			return fmt.Errorf("failed to update proving status for chunk %s: %w", dbChunk.Hash, err)
		}

		log.Info("Inserted chunk", "index", dbChunk.Index, "hash", dbChunk.Hash, "start block", dbChunk.StartBlockNumber, "end block", dbChunk.EndBlockNumber)

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to insert chunk in DB transaction: %w", err)
	}

	// 5.4 Reproduce batch.
	dbParentBatch, err := f.batchORM.GetLatestBatch(f.ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest batch from DB: %w", err)
	}

	var encBatch encoding.Batch
	encBatch.Index = dbParentBatch.Index + 1
	encBatch.ParentBatchHash = common.HexToHash(dbParentBatch.Hash)
	encBatch.TotalL1MessagePoppedBefore = dbChunk.TotalL1MessagesPoppedBefore
	encBatch.PrevL1MessageQueueHash = chunk.PrevL1MessageQueueHash
	encBatch.PostL1MessageQueueHash = chunk.PostL1MessageQueueHash
	encBatch.Chunks = []*encoding.Chunk{&chunk}
	encBatch.Blocks = blocks

	batchMetrics, err := butils.CalculateBatchMetrics(&encBatch, codec.Version(), false)
	if err != nil {
		return fmt.Errorf("failed to calculate batch metrics: %w", err)
	}

	err = f.db.Transaction(func(dbTX *gorm.DB) error {
		dbBatch, err := f.batchORM.InsertBatch(f.ctx, &encBatch, codec.Version(), *batchMetrics, dbTX)
		if err != nil {
			return fmt.Errorf("failed to insert batch to DB: %w", err)
		}
		if err = f.chunkORM.UpdateBatchHashInRange(f.ctx, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex, dbBatch.Hash, dbTX); err != nil {
			return fmt.Errorf("failed to update batch_hash for chunks (batch hash: %s, start chunk: %d, end chunk: %d): %w", dbBatch.Hash, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex, err)
		}

		if err = f.batchORM.UpdateProvingStatus(f.ctx, dbBatch.Hash, types.ProvingTaskVerified, dbTX); err != nil {
			return fmt.Errorf("failed to update proving status for batch %s: %w", dbBatch.Hash, err)
		}
		if err = f.batchORM.UpdateRollupStatusCommitAndFinalizeTxHash(f.ctx, dbBatch.Hash, types.RollupFinalized, batch.commit.TxHash().Hex(), batch.finalize.TxHash().Hex(), dbTX); err != nil {
			return fmt.Errorf("failed to update rollup status for batch %s: %w", dbBatch.Hash, err)
		}

		log.Info("Inserted batch", "index", dbBatch.Index, "hash", dbBatch.Hash, "start chunk", dbBatch.StartChunkIndex, "end chunk", dbBatch.EndChunkIndex)

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to insert batch in DB transaction: %w", err)
	}

	return nil
}

func (f *FullRecovery) getBatchBlockRangeFromBlob(codec encoding.Codec, blobVersionedHash common.Hash, l1BlockTime uint64) ([]encoding.DABlock, error) {
	blob, err := f.beaconNodeClient.GetBlobByVersionedHashAndBlockTime(f.ctx, blobVersionedHash, l1BlockTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob %s: %w", blobVersionedHash.Hex(), err)
	}
	if blob == nil {
		return nil, fmt.Errorf("blob %s not found", blobVersionedHash.Hex())
	}

	blobPayload, err := codec.DecodeBlob(blob)
	if err != nil {
		return nil, fmt.Errorf("blob %s decode error: %w", blobVersionedHash.Hex(), err)
	}

	blocks := blobPayload.Blocks()
	if len(blocks) == 0 {
		return nil, fmt.Errorf("empty blocks in blob %s", blobVersionedHash.Hex())
	}

	return blocks, nil
}

type batchEvents struct {
	commit   *l1.CommitBatchEvent
	finalize *l1.FinalizeBatchEvent
}

func newBatchEvents(commit *l1.CommitBatchEvent, finalize *l1.FinalizeBatchEvent) *batchEvents {
	if commit.BatchIndex().Uint64() > finalize.BatchIndex().Uint64() {
		panic(fmt.Sprintf("commit and finalize batch index mismatch: %d != %d", commit.BatchIndex().Uint64(), finalize.BatchIndex().Uint64()))
	}

	return &batchEvents{
		commit:   commit,
		finalize: finalize,
	}
}

func (e *batchEvents) CompareTo(other *batchEvents) int {
	return e.commit.BatchIndex().Cmp(other.commit.BatchIndex())
}
