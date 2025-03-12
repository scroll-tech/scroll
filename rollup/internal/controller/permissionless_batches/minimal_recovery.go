package permissionless_batches

import (
	"context"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"
	"gorm.io/gorm"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
	"scroll-tech/common/types"

	"scroll-tech/database/migrate"
	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/watcher"
	"scroll-tech/rollup/internal/orm"
)

const (
	// defaultFakeRestoredChunkIndex is the default index of the last restored fake chunk. It is used to be able to generate new chunks pretending that we have already processed some chunks.
	defaultFakeRestoredChunkIndex uint64 = 1337
	// defaultFakeRestoredBundleIndex is the default index of the last restored fake bundle. It is used to be able to generate new bundles pretending that we have already processed some bundles.
	defaultFakeRestoredBundleIndex uint64 = 1
)

type MinimalRecovery struct {
	ctx       context.Context
	cfg       *config.Config
	genesis   *core.Genesis
	db        *gorm.DB
	chunkORM  *orm.Chunk
	batchORM  *orm.Batch
	bundleORM *orm.Bundle

	chunkProposer  *watcher.ChunkProposer
	batchProposer  *watcher.BatchProposer
	bundleProposer *watcher.BundleProposer
	l2Watcher      *watcher.L2WatcherClient
}

func NewRecovery(ctx context.Context, cfg *config.Config, genesis *core.Genesis, db *gorm.DB, chunkProposer *watcher.ChunkProposer, batchProposer *watcher.BatchProposer, bundleProposer *watcher.BundleProposer, l2Watcher *watcher.L2WatcherClient) *MinimalRecovery {
	return &MinimalRecovery{
		ctx:            ctx,
		cfg:            cfg,
		genesis:        genesis,
		db:             db,
		chunkORM:       orm.NewChunk(db),
		batchORM:       orm.NewBatch(db),
		bundleORM:      orm.NewBundle(db),
		chunkProposer:  chunkProposer,
		batchProposer:  batchProposer,
		bundleProposer: bundleProposer,
		l2Watcher:      l2Watcher,
	}
}

func (r *MinimalRecovery) RecoveryNeeded() bool {
	chunk, err := r.chunkORM.GetLatestChunk(r.ctx)
	if err != nil || chunk == nil {
		return true
	}
	if chunk.Index <= defaultFakeRestoredChunkIndex {
		return true
	}

	batch, err := r.batchORM.GetLatestBatch(r.ctx)
	if err != nil {
		return true
	}
	if batch.Index <= r.cfg.RecoveryConfig.LatestFinalizedBatch {
		return true
	}

	bundle, err := r.bundleORM.GetLatestBundle(r.ctx)
	if err != nil {
		return true
	}
	if bundle.Index <= defaultFakeRestoredBundleIndex {
		return true
	}

	return false
}

func (r *MinimalRecovery) Run() error {
	// Make sure we start from a clean state.
	if err := r.resetDB(); err != nil {
		return fmt.Errorf("failed to reset DB: %w", err)
	}

	// Restore minimal previous state required to be able to create new chunks, batches and bundles.
	restoredFinalizedChunk, restoredFinalizedBatch, restoredFinalizedBundle, err := r.restoreMinimalPreviousState()
	if err != nil {
		return fmt.Errorf("failed to restore minimal previous state: %w", err)
	}

	// Fetch and insert the missing blocks from the last block in the latestFinalizedBatch to the latest L2 block.
	fromBlock := restoredFinalizedChunk.EndBlockNumber
	toBlock, err := r.fetchL2Blocks(fromBlock, r.cfg.RecoveryConfig.L2BlockHeightLimit)
	if err != nil {
		return fmt.Errorf("failed to fetch L2 blocks: %w", err)
	}

	// Create chunks for L2 blocks.
	log.Info("Creating chunks for L2 blocks", "from", fromBlock, "to", toBlock)

	var latestChunk *orm.Chunk
	var count int
	for {
		if err = r.chunkProposer.ProposeChunk(); err != nil {
			return fmt.Errorf("failed to propose chunk: %w", err)
		}
		count++

		latestChunk, err = r.chunkORM.GetLatestChunk(r.ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest latestFinalizedChunk: %w", err)
		}

		log.Info("Chunk created", "index", latestChunk.Index, "hash", latestChunk.Hash, "StartBlockNumber", latestChunk.StartBlockNumber, "EndBlockNumber", latestChunk.EndBlockNumber, "TotalL1MessagesPoppedBefore", latestChunk.TotalL1MessagesPoppedBefore)

		// We have created chunks for all available L2 blocks.
		if latestChunk.EndBlockNumber >= toBlock {
			break
		}
	}

	log.Info("Chunks created", "count", count, "latest Chunk", latestChunk.Index, "hash", latestChunk.Hash, "StartBlockNumber", latestChunk.StartBlockNumber, "EndBlockNumber", latestChunk.EndBlockNumber, "TotalL1MessagesPoppedBefore", latestChunk.TotalL1MessagesPoppedBefore, "PrevL1MessageQueueHash", latestChunk.PrevL1MessageQueueHash, "PostL1MessageQueueHash", latestChunk.PostL1MessageQueueHash)

	// Create batch for the created chunks. We only allow 1 batch it needs to be submitted (and finalized) with a proof in a single step.
	log.Info("Creating batch for chunks", "from", restoredFinalizedChunk.Index+1, "to", latestChunk.Index)

	r.batchProposer.TryProposeBatch()
	latestBatch, err := r.batchORM.GetLatestBatch(r.ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest latestFinalizedBatch: %w", err)
	}

	// Sanity check that the batch was created correctly:
	// 1. should be a new batch
	// 2. should contain all chunks created
	if restoredFinalizedBatch.Index+1 != latestBatch.Index {
		return fmt.Errorf("batch was not created correctly, expected %d but got %d", restoredFinalizedBatch.Index+1, latestBatch.Index)
	}

	firstChunkInBatch, err := r.chunkORM.GetChunkByIndex(r.ctx, latestBatch.StartChunkIndex)
	if err != nil {
		return fmt.Errorf("failed to get first chunk in batch: %w", err)
	}
	lastChunkInBatch, err := r.chunkORM.GetChunkByIndex(r.ctx, latestBatch.EndChunkIndex)
	if err != nil {
		return fmt.Errorf("failed to get last chunk in batch: %w", err)
	}

	// Make sure that the batch contains all previously created chunks and thus all blocks. If not the user will need to
	// produce another batch (running the application again) starting from the end block of the last chunk in the batch + 1.
	if latestBatch.EndChunkIndex != latestChunk.Index {
		log.Warn("Produced batch does not contain all chunks and blocks. You'll need to produce another batch starting from end block+1.", "starting block", firstChunkInBatch.StartBlockNumber, "end block", lastChunkInBatch.EndBlockNumber, "latest block", latestChunk.EndBlockNumber)
	}

	log.Info("Batch created", "index", latestBatch.Index, "hash", latestBatch.Hash, "StartChunkIndex", latestBatch.StartChunkIndex, "EndChunkIndex", latestBatch.EndChunkIndex, "starting block", firstChunkInBatch.StartBlockNumber, "ending block", lastChunkInBatch.EndBlockNumber, "PrevL1MessageQueueHash", latestBatch.PrevL1MessageQueueHash, "PostL1MessageQueueHash", latestBatch.PostL1MessageQueueHash)

	if err = r.bundleProposer.UpdateDBBundleInfo([]*orm.Batch{latestBatch}, encoding.CodecVersion(latestBatch.CodecVersion)); err != nil {
		return fmt.Errorf("failed to create bundle: %w", err)
	}

	latestBundle, err := r.bundleORM.GetLatestBundle(r.ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest bundle: %w", err)
	}

	// Sanity check that the bundle was created correctly:
	// 1. should be a new bundle
	// 2. should only contain 1 batch, the one we created
	if restoredFinalizedBundle.Index == latestBundle.Index {
		return fmt.Errorf("bundle was not created correctly")
	}
	if latestBundle.StartBatchIndex != latestBatch.Index || latestBundle.EndBatchIndex != latestBatch.Index {
		return fmt.Errorf("bundle does not contain the correct batch: %d != %d", latestBundle.StartBatchIndex, latestBatch.Index)
	}

	log.Info("Bundle created", "index", latestBundle.Index, "hash", latestBundle.Hash, "StartBatchIndex", latestBundle.StartBatchIndex, "EndBatchIndex", latestBundle.EndBatchIndex, "starting block", firstChunkInBatch.StartBlockNumber, "ending block", lastChunkInBatch.EndBlockNumber)

	return nil
}

// restoreMinimalPreviousState restores the minimal previous state required to be able to create new chunks, batches and bundles.
func (r *MinimalRecovery) restoreMinimalPreviousState() (*orm.Chunk, *orm.Batch, *orm.Bundle, error) {
	log.Info("Restoring previous state with", "L1 block height", r.cfg.RecoveryConfig.L1BlockHeight, "latest finalized batch", r.cfg.RecoveryConfig.LatestFinalizedBatch)

	l1Client, err := ethclient.Dial(r.cfg.L1Config.Endpoint)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to L1 client: %w", err)
	}
	reader, err := l1.NewReader(r.ctx, l1.Config{
		ScrollChainAddress:    r.genesis.Config.Scroll.L1Config.ScrollChainAddress,
		L1MessageQueueAddress: r.genesis.Config.Scroll.L1Config.L1MessageQueueV2Address,
	}, l1Client)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create L1 reader: %w", err)
	}

	// 1. Sanity check user input: Make sure that the user's L1 block height is not higher than the latest finalized block number.
	latestFinalizedL1Block, err := reader.GetLatestFinalizedBlockNumber()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get latest finalized L1 block number: %w", err)
	}
	if r.cfg.RecoveryConfig.L1BlockHeight > latestFinalizedL1Block {
		return nil, nil, nil, fmt.Errorf("specified L1 block height is higher than the latest finalized block number: %d > %d", r.cfg.RecoveryConfig.L1BlockHeight, latestFinalizedL1Block)
	}

	log.Info("Latest finalized L1 block number", "latest finalized L1 block", latestFinalizedL1Block)

	// 2. Make sure that the specified batch is indeed finalized on the L1 rollup contract and is the latest finalized batch.
	var latestFinalizedBatchIndex uint64
	if r.cfg.RecoveryConfig.ForceLatestFinalizedBatch {
		latestFinalizedBatchIndex = r.cfg.RecoveryConfig.LatestFinalizedBatch
	} else {
		latestFinalizedBatchIndex, err = reader.LatestFinalizedBatchIndex(latestFinalizedL1Block)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get latest finalized batch: %w", err)
		}
		if r.cfg.RecoveryConfig.LatestFinalizedBatch != latestFinalizedBatchIndex {
			return nil, nil, nil, fmt.Errorf("batch %d is not the latest finalized batch: %d", r.cfg.RecoveryConfig.LatestFinalizedBatch, latestFinalizedBatchIndex)
		}
	}

	// Find the commit event for the latest finalized batch.
	var batchCommitEvent *l1.CommitBatchEvent
	err = reader.FetchRollupEventsInRangeWithCallback(r.cfg.RecoveryConfig.L1BlockHeight, latestFinalizedL1Block, func(event l1.RollupEvent) bool {
		if event.Type() == l1.CommitEventType && event.BatchIndex().Uint64() == latestFinalizedBatchIndex {
			batchCommitEvent = event.(*l1.CommitBatchEvent)
			// We found the commit event for the batch, stop searching.
			return false
		}

		// Continue until we find the commit event for the batch.
		return true
	})
	if batchCommitEvent == nil {
		return nil, nil, nil, fmt.Errorf("commit event not found for batch %d", latestFinalizedBatchIndex)
	}

	log.Info("Found commit event for batch", "batch", batchCommitEvent.BatchIndex(), "hash", batchCommitEvent.BatchHash(), "L1 block height", batchCommitEvent.BlockNumber(), "L1 tx hash", batchCommitEvent.TxHash())

	// 3. Fetch commit tx data for latest finalized batch and decode it.
	daBatch, daBlobPayload, err := r.decodeLatestFinalizedBatch(reader, batchCommitEvent)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode latest finalized batch: %w", err)
	}
	fmt.Println(daBatch, daBlobPayload)

	blocksInBatch := daBlobPayload.Blocks()

	if len(blocksInBatch) == 0 {
		return nil, nil, nil, fmt.Errorf("no blocks in batch %d", batchCommitEvent.BatchIndex())
	}
	lastBlockInBatch := blocksInBatch[len(blocksInBatch)-1]

	log.Info("Last L2 block in batch", "batch", batchCommitEvent.BatchIndex(), "L2 block", lastBlockInBatch, "PostL1MessageQueueHash", daBlobPayload.PostL1MessageQueueHash())

	// 4. Get the L1 messages count after the latest finalized batch.
	var l1MessagesCount uint64
	if r.cfg.RecoveryConfig.ForceL1MessageCount == 0 {
		l1MessagesCount, err = reader.FinalizedL1MessageQueueIndex(latestFinalizedL1Block)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get L1 messages count: %w", err)
		}
	} else {
		l1MessagesCount = r.cfg.RecoveryConfig.ForceL1MessageCount
	}

	log.Info("L1 messages count after latest finalized batch", "batch", batchCommitEvent.BatchIndex(), "count", l1MessagesCount)

	// 5. Insert minimal state to DB.
	chunk, err := r.chunkORM.InsertPermissionlessChunk(r.ctx, defaultFakeRestoredChunkIndex, daBatch.Version(), daBlobPayload, l1MessagesCount)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to insert chunk raw: %w", err)
	}

	log.Info("Inserted last finalized chunk to DB", "chunk", chunk.Index, "hash", chunk.Hash, "StartBlockNumber", chunk.StartBlockNumber, "EndBlockNumber", chunk.EndBlockNumber, "TotalL1MessagesPoppedBefore", chunk.TotalL1MessagesPoppedBefore)

	batch, err := r.batchORM.InsertPermissionlessBatch(r.ctx, batchCommitEvent.BatchIndex(), batchCommitEvent.BatchHash(), daBatch.Version(), chunk)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to insert batch raw: %w", err)
	}

	log.Info("Inserted last finalized batch to DB", "batch", batch.Index, "hash", batch.Hash)

	var bundle *orm.Bundle
	err = r.db.Transaction(func(dbTX *gorm.DB) error {
		bundle, err = r.bundleORM.InsertBundle(r.ctx, []*orm.Batch{batch}, encoding.CodecVersion(batch.CodecVersion), dbTX)
		if err != nil {
			return fmt.Errorf("failed to insert bundle: %w", err)
		}
		if err = r.bundleORM.UpdateProvingStatus(r.ctx, bundle.Hash, types.ProvingTaskVerified, dbTX); err != nil {
			return fmt.Errorf("failed to update proving status: %w", err)
		}
		if err = r.bundleORM.UpdateRollupStatus(r.ctx, bundle.Hash, types.RollupFinalized); err != nil {
			return fmt.Errorf("failed to update rollup status: %w", err)
		}

		log.Info("Inserted last finalized bundle to DB", "bundle", bundle.Index, "hash", bundle.Hash, "StartBatchIndex", bundle.StartBatchIndex, "EndBatchIndex", bundle.EndBatchIndex)

		return nil
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to insert bundle: %w", err)
	}
	return chunk, batch, bundle, nil
}

func (r *MinimalRecovery) decodeLatestFinalizedBatch(reader *l1.Reader, event *l1.CommitBatchEvent) (encoding.DABatch, encoding.DABlobPayload, error) {
	blockHeader, err := reader.FetchBlockHeaderByNumber(event.BlockNumber())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get header by number, err: %w", err)
	}

	args, err := reader.FetchCommitTxData(event)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch commit tx data: %w", err)
	}

	codecVersion := encoding.CodecVersion(args.Version)
	if codecVersion < encoding.CodecV7 {
		return nil, nil, fmt.Errorf("codec version %d is not supported", codecVersion)
	}

	codec, err := encoding.CodecFromVersion(codecVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get codec: %w", err)
	}

	// Since we only store the last batch hash committed in a single tx in the contracts we can also only ever
	// finalize a last batch of a tx. This means we can assume here that the batch given in the event is the last batch
	// that was committed in the tx.

	if event.BatchIndex().Uint64()+1 < uint64(len(args.BlobHashes)) {
		return nil, nil, fmt.Errorf("batch index %d+1 is lower than the number of blobs %d", event.BatchIndex().Uint64(), len(args.BlobHashes))
	}
	firstBatchIndex := event.BatchIndex().Uint64() + 1 - uint64(len(args.BlobHashes))

	var targetBatch encoding.DABatch
	var targetBlobVersionedHash common.Hash
	parentBatchHash := args.ParentBatchHash
	for i, blobVersionedHash := range args.BlobHashes {
		batchIndex := firstBatchIndex + uint64(i)

		calculatedBatch, err := codec.NewDABatchFromParams(batchIndex, blobVersionedHash, parentBatchHash)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create new DA batch from params, batch index: %d, err: %w", event.BatchIndex().Uint64(), err)
		}
		parentBatchHash = calculatedBatch.Hash()

		if batchIndex == event.BatchIndex().Uint64() {
			if calculatedBatch.Hash() != event.BatchHash() {
				return nil, nil, fmt.Errorf("batch hash mismatch for batch %d, expected: %s, got: %s", event.BatchIndex(), event.BatchHash().String(), calculatedBatch.Hash().String())
			}
			// We found the batch we are looking for, break out of the loop.
			targetBatch = calculatedBatch
			targetBlobVersionedHash = blobVersionedHash
			break
		}
	}

	if targetBatch == nil {
		return nil, nil, fmt.Errorf("target batch with index %d could not be found and decoded", event.BatchIndex())
	}

	// sanity check that this is indeed the last batch in the tx
	if targetBatch.Hash() != args.LastBatchHash {
		return nil, nil, fmt.Errorf("last batch hash mismatch for batch %d, expected: %s, got: %s", event.BatchIndex(), args.LastBatchHash.String(), targetBatch.Hash().String())
	}

	// TODO: add support for multiple blob clients
	blobClient := blob_client.NewBlobClients()
	if r.cfg.RecoveryConfig.L1BeaconNodeEndpoint != "" {
		client, err := blob_client.NewBeaconNodeClient(r.cfg.RecoveryConfig.L1BeaconNodeEndpoint)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create beacon node client: %w", err)
		}
		blobClient.AddBlobClient(client)
	}

	blob, err := blobClient.GetBlobByVersionedHashAndBlockTime(r.ctx, targetBlobVersionedHash, blockHeader.Time)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get blob by versioned hash and block time for batch %d: %w", event.BatchIndex(), err)
	}

	daBlobPayload, err := codec.DecodeBlob(blob)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode blob for batch %d: %w", event.BatchIndex(), err)
	}

	return targetBatch, daBlobPayload, nil
}

func (r *MinimalRecovery) fetchL2Blocks(fromBlock uint64, l2BlockHeightLimit uint64) (uint64, error) {
	if l2BlockHeightLimit > 0 && fromBlock > l2BlockHeightLimit {
		return 0, fmt.Errorf("fromBlock (latest finalized L2 block) is higher than specified L2BlockHeightLimit: %d > %d", fromBlock, l2BlockHeightLimit)
	}

	log.Info("Fetching L2 blocks with", "fromBlock", fromBlock, "l2BlockHeightLimit", l2BlockHeightLimit)

	// Fetch and insert the missing blocks from the last block in the batch to the latest L2 block.
	latestL2Block, err := r.l2Watcher.Client.BlockNumber(r.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest L2 block number: %w", err)
	}

	log.Info("Latest L2 block number", "latest L2 block", latestL2Block)

	if l2BlockHeightLimit > latestL2Block {
		return 0, fmt.Errorf("l2BlockHeightLimit is higher than the latest L2 block number, not all blocks are available in L2geth: %d > %d", l2BlockHeightLimit, latestL2Block)
	}

	toBlock := latestL2Block
	if l2BlockHeightLimit > 0 {
		toBlock = l2BlockHeightLimit
	}

	err = r.l2Watcher.GetAndStoreBlocks(r.ctx, fromBlock, toBlock)
	if err != nil {
		return 0, fmt.Errorf("failed to get and store blocks: %w", err)
	}

	log.Info("Fetched L2 blocks from", "fromBlock", fromBlock, "toBlock", toBlock)

	return toBlock, nil
}

func (r *MinimalRecovery) resetDB() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get db connection: %w", err)
	}

	// reset and init DB
	var v int64
	err = migrate.Rollback(sqlDB, &v)
	if err != nil {
		return fmt.Errorf("failed to rollback db: %w", err)
	}

	err = migrate.Migrate(sqlDB)
	if err != nil {
		return fmt.Errorf("failed to migrate db: %w", err)
	}

	return nil
}
