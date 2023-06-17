package watcher

import (
	"context"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
)

type BatchProposer struct {
	ctx context.Context
	db  *gorm.DB

	batchOrm *orm.Batch
	chunkOrm *orm.Chunk
	l2Block  *orm.L2Block

	batchTimeoutSec         uint64
	maxCalldataGasPerChunk  uint64
	maxCalldataSizePerBatch uint64
	minCalldataSizePerBatch uint64
}

func NewBatchProposer(ctx context.Context, cfg *config.BatchProposerConfig, db *gorm.DB) *BatchProposer {
	return &BatchProposer{
		ctx:                     ctx,
		db:                      db,
		batchOrm:                orm.NewBatch(db),
		chunkOrm:                orm.NewChunk(db),
		l2Block:                 orm.NewL2Block(db),
		batchTimeoutSec:         cfg.BatchTimeoutSec,
		maxCalldataGasPerChunk:  cfg.MaxCalldataGasPerChunk,
		maxCalldataSizePerBatch: cfg.MaxCalldataSizePerBatch,
		minCalldataSizePerBatch: cfg.MinCalldataSizePerBatch,
	}
}

func (p *BatchProposer) TryProposeBatch() {
	batchChunks, err := p.proposeBatchChunks()
	if err != nil {
		log.Error("proposeBatch failed", "err", err)
		return
	}
	if err := p.batchOrm.InsertBatch(p.ctx, batchChunks, p.chunkOrm); err != nil {
		log.Error("InsertBatch failed", "err", err)
	}
}

func (p *BatchProposer) proposeBatchChunks() ([]*bridgeTypes.Chunk, error) {
	dbChunks, err := p.chunkOrm.GetUnbatchedChunks(p.ctx)
	if err != nil {
		return nil, err
	}

	if len(dbChunks) == 0 {
		log.Warn("No Unbatched Chunks")
		return nil, nil
	}

	firstChunk := dbChunks[0]
	totalPayloadSize := firstChunk.TotalPayloadSize

	if totalPayloadSize > p.maxPayloadSizePerBatch {
		log.Warn("The first chunk exceeds the max payload size limit",
			"total payload size", totalPayloadSize,
			"max payload size limit", p.maxPayloadSizePerBatch,
		)
		return p.dbChunksToBridgeChunks(dbChunks[:1])
	}

	for i, chunk := range dbChunks[1:] {
		totalPayloadSize += chunk.TotalPayloadSize
		if totalPayloadSize > p.maxPayloadSizePerBatch {
			return p.dbChunksToBridgeChunks(dbChunks[:i+1])
		}
	}

	var hasChunkTimeout bool
	currentTimeSec := uint64(time.Now().Unix())
	earliestBlockTime, err := p.l2Block.GetBlockTimestamp(dbChunks[0].StartBlockNumber)
	if err != nil {
		log.Error("GetBlockTimestamp failed", "block number", dbChunks[0].StartBlockNumber, "err", err)
		return nil, err
	}
	if earliestBlockTime+p.batchTimeoutSec > currentTimeSec {
		log.Warn("first block timeout", "block number", dbChunks[0].StartBlockNumber, "block timestamp", earliestBlockTime, "chunk time limit", currentTimeSec)
		hasChunkTimeout = true
	}

	if !hasChunkTimeout && totalPayloadSize < p.minPayloadSizePerBatch {
		log.Warn("The payload size of the batch is less than the minimum limit",
			"totalPayloadSize", totalPayloadSize,
			"minPayloadSizePerBatch", p.minPayloadSizePerBatch,
		)
		return nil, nil
	}
	return p.dbChunksToBridgeChunks(dbChunks)
}

func (p *BatchProposer) dbChunksToBridgeChunks(dbChunks []*orm.Chunk) ([]*bridgeTypes.Chunk, error) {
	chunks := make([]*bridgeTypes.Chunk, len(dbChunks))
	for i, c := range dbChunks {
		wrappedBlocks, err := p.l2Block.GetL2BlocksInClosedRange(p.ctx, c.StartBlockNumber, c.EndBlockNumber)
		if err != nil {
			log.Error("Failed to fetch wrapped blocks", "error", err)
			return nil, err
		}
		chunks[i] = &bridgeTypes.Chunk{
			Blocks: wrappedBlocks,
		}
	}
	return chunks, nil
}
