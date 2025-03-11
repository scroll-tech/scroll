package relayer

import (
	"context"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
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

	chunkProposer  *watcher.ChunkProposer
	batchProposer  *watcher.BatchProposer
	bundleProposer *watcher.BundleProposer
	l2Watcher      *watcher.L2WatcherClient
	l1Client       *ethclient.Client
	l1Reader       *l1.Reader
}

func NewFullRecovery(ctx context.Context, cfg *config.Config, genesis *core.Genesis, db *gorm.DB, chunkProposer *watcher.ChunkProposer, batchProposer *watcher.BatchProposer, bundleProposer *watcher.BundleProposer, l2Watcher *watcher.L2WatcherClient, l1Client *ethclient.Client, l1Reader *l1.Reader) *FullRecovery {
	return &FullRecovery{
		ctx:       ctx,
		cfg:       cfg,
		genesis:   genesis,
		db:        db,
		blockORM:  orm.NewL2Block(db),
		chunkORM:  orm.NewChunk(db),
		batchORM:  orm.NewBatch(db),
		bundleORM: orm.NewBundle(db),

		chunkProposer:  chunkProposer,
		batchProposer:  batchProposer,
		bundleProposer: bundleProposer,
		l2Watcher:      l2Watcher,
		l1Client:       l1Client,
		l1Reader:       l1Reader,
	}
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
	receipt, err := f.l1Client.TransactionReceipt(f.ctx, common.HexToHash(latestDBBatch.CommitTxHash))
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt of latest DB batch finalization transaction: %w", err)
	}
	fromBlock := receipt.BlockNumber.Uint64()

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

			// with bundles all commited batches until this finalized batch are finalized in the same bundle
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

		case l1.RevertEventType:
			// We ignore reverted batches.
			commitsHeapMap.RemoveByKey(event.BatchIndex().Uint64())
		}

		return true
	})
	if err != nil {
		return fmt.Errorf("failed to fetch rollup events: %w", err)
	}

	// 5. Process all finalized batches: fetch L2 blocks and reproduce chunks and batches.
	for batchEventsHeap.Len() > 0 {
		nextBatch := batchEventsHeap.Pop().Value()
		if err = f.processFinalizedBatch(nextBatch); err != nil {
			return fmt.Errorf("failed to process finalized batch %d %s: %w", nextBatch.commit.BatchIndex(), nextBatch.commit.BatchHash(), err)
		}

		log.Info("Processed finalized batch", "batch", nextBatch.commit.BatchIndex(), "hash", nextBatch.commit.BatchHash())
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

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to insert bundle in DB transaction: %w", err)
		}
	}

	return nil
}

func (f *FullRecovery) processFinalizedBatch(nextBatch *batchEvents) error {
	log.Info("Processing finalized batch", "batch", nextBatch.commit.BatchIndex(), "hash", nextBatch.commit.BatchHash())

	// 5.1. Fetch commit tx data for batch (via commit event).
	args, err := f.l1Reader.FetchCommitTxData(nextBatch.commit)
	if err != nil {
		return fmt.Errorf("failed to fetch commit tx data: %w", err)
	}

	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(args.Version))
	if err != nil {
		return fmt.Errorf("failed to get codec: %w", err)
	}

	daChunksRawTxs, err := codec.DecodeDAChunksRawTx(args.Chunks)
	if err != nil {
		return fmt.Errorf("failed to decode DA chunks: %w", err)
	}
	lastChunk := daChunksRawTxs[len(daChunksRawTxs)-1]
	lastBlockInBatch := lastChunk.Blocks[len(lastChunk.Blocks)-1].Number()

	log.Info("Fetching L2 blocks from l2geth", "batch", nextBatch.commit.BatchIndex(), "last L2 block in batch", lastBlockInBatch)

	// 5.2. Fetch L2 blocks for the entire batch.
	if err = f.l2Watcher.TryFetchRunningMissingBlocks(lastBlockInBatch); err != nil {
		return fmt.Errorf("failed to fetch L2 blocks: %w", err)
	}

	// 5.3. Reproduce chunks.
	daChunks := make([]*encoding.Chunk, 0, len(daChunksRawTxs))
	dbChunks := make([]*orm.Chunk, 0, len(daChunksRawTxs))
	for _, daChunkRawTxs := range daChunksRawTxs {
		start := daChunkRawTxs.Blocks[0].Number()
		end := daChunkRawTxs.Blocks[len(daChunkRawTxs.Blocks)-1].Number()

		blocks, err := f.blockORM.GetL2BlocksInRange(f.ctx, start, end)
		if err != nil {
			return fmt.Errorf("failed to get L2 blocks in range: %w", err)
		}

		log.Info("Reproducing chunk", "start block", start, "end block", end)

		var chunk encoding.Chunk
		for _, block := range blocks {
			chunk.Blocks = append(chunk.Blocks, block)
		}

		metrics, err := butils.CalculateChunkMetrics(&chunk, codec.Version())
		if err != nil {
			return fmt.Errorf("failed to calculate chunk metrics: %w", err)
		}

		err = f.db.Transaction(func(dbTX *gorm.DB) error {
			dbChunk, err := f.chunkORM.InsertChunk(f.ctx, &chunk, codec.Version(), *metrics, dbTX)
			if err != nil {
				return fmt.Errorf("failed to insert chunk to DB: %w", err)
			}
			if err := f.blockORM.UpdateChunkHashInRange(f.ctx, dbChunk.StartBlockNumber, dbChunk.EndBlockNumber, dbChunk.Hash, dbTX); err != nil {
				return fmt.Errorf("failed to update chunk_hash for l2_blocks (chunk hash: %s, start block: %d, end block: %d): %w", dbChunk.Hash, dbChunk.StartBlockNumber, dbChunk.EndBlockNumber, err)
			}

			if err = f.chunkORM.UpdateProvingStatus(f.ctx, dbChunk.Hash, types.ProvingTaskVerified, dbTX); err != nil {
				return fmt.Errorf("failed to update proving status for chunk %s: %w", dbChunk.Hash, err)
			}

			daChunks = append(daChunks, &chunk)
			dbChunks = append(dbChunks, dbChunk)

			log.Info("Inserted chunk", "index", dbChunk.Index, "hash", dbChunk.Hash, "start block", dbChunk.StartBlockNumber, "end block", dbChunk.EndBlockNumber)

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to insert chunk in DB transaction: %w", err)
		}
	}

	// 5.4 Reproduce batch.
	dbParentBatch, err := f.batchORM.GetLatestBatch(f.ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest batch from DB: %w", err)
	}

	var batch encoding.Batch
	batch.Index = dbParentBatch.Index + 1
	batch.ParentBatchHash = common.HexToHash(dbParentBatch.Hash)
	batch.TotalL1MessagePoppedBefore = dbChunks[0].TotalL1MessagesPoppedBefore

	for _, chunk := range daChunks {
		batch.Chunks = append(batch.Chunks, chunk)
	}

	metrics, err := butils.CalculateBatchMetrics(&batch, codec.Version())
	if err != nil {
		return fmt.Errorf("failed to calculate batch metrics: %w", err)
	}

	err = f.db.Transaction(func(dbTX *gorm.DB) error {
		dbBatch, err := f.batchORM.InsertBatch(f.ctx, &batch, codec.Version(), *metrics, dbTX)
		if err != nil {
			return fmt.Errorf("failed to insert batch to DB: %w", err)
		}
		if err = f.chunkORM.UpdateBatchHashInRange(f.ctx, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex, dbBatch.Hash, dbTX); err != nil {
			return fmt.Errorf("failed to update batch_hash for chunks (batch hash: %s, start chunk: %d, end chunk: %d): %w", dbBatch.Hash, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex, err)
		}

		if err = f.batchORM.UpdateProvingStatus(f.ctx, dbBatch.Hash, types.ProvingTaskVerified, dbTX); err != nil {
			return fmt.Errorf("failed to update proving status for batch %s: %w", dbBatch.Hash, err)
		}
		if err = f.batchORM.UpdateRollupStatusCommitAndFinalizeTxHash(f.ctx, dbBatch.Hash, types.RollupFinalized, nextBatch.commit.TxHash().Hex(), nextBatch.finalize.TxHash().Hex(), dbTX); err != nil {
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
