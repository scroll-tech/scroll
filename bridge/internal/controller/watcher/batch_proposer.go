package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/orm"
)

// BatchProposer proposes batches based on available unbatched chunks.
type BatchProposer struct {
	ctx context.Context
	db  *gorm.DB

	batchOrm *orm.Batch
	chunkOrm *orm.Chunk
	l2Block  *orm.L2Block

	maxChunkNumPerBatch             uint64
	maxL1CommitGasPerBatch          uint64
	maxL1CommitCalldataSizePerBatch uint32
	minChunkNumPerBatch             uint64
	batchTimeoutSec                 uint64
}

// NewBatchProposer creates a new BatchProposer instance.
func NewBatchProposer(ctx context.Context, cfg *config.BatchProposerConfig, db *gorm.DB) *BatchProposer {
	return &BatchProposer{
		ctx:                             ctx,
		db:                              db,
		batchOrm:                        orm.NewBatch(db),
		chunkOrm:                        orm.NewChunk(db),
		l2Block:                         orm.NewL2Block(db),
		maxChunkNumPerBatch:             cfg.MaxChunkNumPerBatch,
		maxL1CommitGasPerBatch:          cfg.MaxL1CommitGasPerBatch,
		maxL1CommitCalldataSizePerBatch: cfg.MaxL1CommitCalldataSizePerBatch,
		minChunkNumPerBatch:             cfg.MinChunkNumPerBatch,
		batchTimeoutSec:                 cfg.BatchTimeoutSec,
	}
}

// TryProposeBatch tries to propose a new batches.
func (p *BatchProposer) TryProposeBatch() {
	dbChunks, err := p.proposeBatchChunks()
	if err != nil {
		log.Error("proposeBatchChunks failed", "err", err)
		return
	}
	if err := p.updateBatchInfoInDB(dbChunks); err != nil {
		log.Error("update batch info in db failed", "err", err)
	}
}

func (p *BatchProposer) updateBatchInfoInDB(dbChunks []*orm.Chunk) error {
	numChunks := len(dbChunks)
	if numChunks <= 0 {
		return nil
	}
	chunks, err := p.dbChunksToBridgeChunks(dbChunks)
	if err != nil {
		return err
	}

	startChunkIndex := dbChunks[0].Index
	startChunkHash := dbChunks[0].Hash
	endChunkIndex := dbChunks[numChunks-1].Index
	endChunkHash := dbChunks[numChunks-1].Hash
	err = p.db.Transaction(func(dbTX *gorm.DB) error {
		batch, dbErr := p.batchOrm.InsertBatch(p.ctx, startChunkIndex, endChunkIndex, startChunkHash, endChunkHash, chunks, dbTX)
		if dbErr != nil {
			return dbErr
		}
		dbErr = p.chunkOrm.UpdateBatchHashInRange(p.ctx, startChunkIndex, endChunkIndex, batch.Hash, dbTX)
		if dbErr != nil {
			return dbErr
		}
		return nil
	})
	return err
}

func (p *BatchProposer) proposeBatchChunks() ([]*orm.Chunk, error) {
	dbChunks, err := p.chunkOrm.GetUnbatchedChunks(p.ctx)
	if err != nil {
		return nil, err
	}

	if len(dbChunks) == 0 {
		log.Warn("No Unbatched Chunks")
		return nil, nil
	}

	firstChunk := dbChunks[0]
	totalL1CommitCalldataSize := firstChunk.TotalL1CommitCalldataSize
	totalL1CommitGas := firstChunk.TotalL1CommitGas
	var totalChunks uint64 = 1

	// Check if the first chunk breaks hard limits.
	// If so, it indicates there are bugs in chunk-proposer, manual fix is needed.
	if totalL1CommitGas > p.maxL1CommitGasPerBatch {
		return nil, fmt.Errorf(
			"the first chunk exceeds l1 commit gas limit; start block number: %v, end block number: %v, commit gas: %v, max commit gas limit: %v",
			firstChunk.StartBlockNumber,
			firstChunk.EndBlockNumber,
			totalL1CommitGas,
			p.maxL1CommitGasPerBatch,
		)
	}

	if totalL1CommitCalldataSize > p.maxL1CommitCalldataSizePerBatch {
		return nil, fmt.Errorf(
			"the first chunk exceeds l1 commit calldata size limit; start block number: %v, end block number %v, calldata size: %v, max calldata size limit: %v",
			firstChunk.StartBlockNumber,
			firstChunk.EndBlockNumber,
			totalL1CommitCalldataSize,
			p.maxL1CommitCalldataSizePerBatch,
		)
	}

	for i, chunk := range dbChunks[1:] {
		totalChunks++
		totalL1CommitCalldataSize += chunk.TotalL1CommitCalldataSize
		totalL1CommitGas += chunk.TotalL1CommitGas
		if totalChunks > p.maxChunkNumPerBatch ||
			totalL1CommitCalldataSize > p.maxL1CommitCalldataSizePerBatch ||
			totalL1CommitGas > p.maxL1CommitGasPerBatch {
			return dbChunks[:i+1], nil
		}
	}

	var hasChunkTimeout bool
	currentTimeSec := uint64(time.Now().Unix())
	if dbChunks[0].StartBlockTime+p.batchTimeoutSec < currentTimeSec {
		log.Warn("first block timeout",
			"start block number", dbChunks[0].StartBlockNumber,
			"first block timestamp", dbChunks[0].StartBlockTime,
			"chunk outdated time threshold", currentTimeSec,
		)
		hasChunkTimeout = true
	}

	if !hasChunkTimeout && uint64(len(dbChunks)) < p.minChunkNumPerBatch {
		log.Warn("The payload size of the batch is less than the minimum limit",
			"chunk num", len(dbChunks), "minChunkNumPerBatch", p.minChunkNumPerBatch,
		)
		return nil, nil
	}
	return dbChunks, nil
}

func (p *BatchProposer) dbChunksToBridgeChunks(dbChunks []*orm.Chunk) ([]*types.Chunk, error) {
	chunks := make([]*types.Chunk, len(dbChunks))
	for i, c := range dbChunks {
		wrappedBlocks, err := p.l2Block.GetL2BlocksInRange(p.ctx, c.StartBlockNumber, c.EndBlockNumber)
		if err != nil {
			log.Error("Failed to fetch wrapped blocks",
				"start number", c.StartBlockNumber, "end number", c.EndBlockNumber, "error", err)
			return nil, err
		}
		chunks[i] = &types.Chunk{
			Blocks: wrappedBlocks,
		}
	}
	return chunks, nil
}
