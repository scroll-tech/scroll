package watcher

import (
	"context"
	"errors"
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
		return nil, nil
	}

	firstChunk := dbChunks[0]
	totalL1CommitCalldataSize := firstChunk.TotalL1CommitCalldataSize
	totalL1CommitGas := firstChunk.TotalL1CommitGas
	totalChunks := uint64(1)
	totalL1MessagePopped := firstChunk.TotalL1MessagesPoppedBefore + uint64(firstChunk.TotalL1MessagesPoppedInChunk)

	parentBatch, err := p.batchOrm.GetLatestBatch(p.ctx)
	if err != nil && !errors.Is(errors.Unwrap(err), gorm.ErrRecordNotFound) {
		return nil, err
	}

	getKeccakGas := func(size uint64) uint64 {
		return 30 + 6*((size+31)/32) // 30 + 6 * ceil(size / 32)
	}

	// Add extra gas costs
	totalL1CommitGas += 4 * 2100 // 4 one-time cold sload for commitBatch
	totalL1CommitGas += 20000    // 1 time sstore
	// adjusting gas:
	// add 1 time cold sload (2100 gas) for L1MessageQueue
	// add 1 time cold address access (2600 gas) for L1MessageQueue
	// minus 1 time warm sload (100 gas) & 1 time warm address access (100 gas)
	totalL1CommitGas += (2100 + 2600 - 100 - 100)
	totalL1CommitGas += getKeccakGas(32 * totalChunks) // batch data hash
	if parentBatch != nil {                            // parent batch header hash
		totalL1CommitGas += getKeccakGas(uint64(len(parentBatch.BatchHeader)))
	}
	// batch header size: 89 + 32 * ceil(l1MessagePopped / 256)
	totalL1CommitGas += getKeccakGas(89 + 32*(totalL1MessagePopped+255)/256)

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
		totalL1CommitCalldataSize += chunk.TotalL1CommitCalldataSize
		totalL1CommitGas += chunk.TotalL1CommitGas
		// adjust batch data hash gas cost: add one chunk
		totalL1CommitGas -= getKeccakGas(32 * totalChunks)
		totalChunks++
		totalL1CommitGas += getKeccakGas(32 * totalChunks)
		// adjust batch header hash gas cost: adjust totalL1MessagePopped in calculating header length
		totalL1CommitGas -= getKeccakGas(89 + 32*(totalL1MessagePopped+255)/256)
		totalL1MessagePopped += uint64(chunk.TotalL1MessagesPoppedInChunk)
		totalL1CommitGas += getKeccakGas(89 + 32*(totalL1MessagePopped+255)/256)
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
		log.Warn("The chunk number of the batch is less than the minimum limit",
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
