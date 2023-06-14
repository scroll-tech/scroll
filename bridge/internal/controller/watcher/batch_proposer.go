package watcher

import (
	"context"
	"errors"
	"fmt"

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

	maxPayloadSizePerBatch uint64
	minPayloadSizePerBatch uint64
	maxChunkNumPerBatch    uint64
}

func NewBatchProposer(ctx context.Context, cfg *config.BatchProposerConfig, db *gorm.DB) *BatchProposer {
	return &BatchProposer{
		ctx:                    ctx,
		db:                     db,
		batchOrm:               orm.NewBatch(db),
		chunkOrm:               orm.NewChunk(db),
		l2Block:                orm.NewL2Block(db),
		maxPayloadSizePerBatch: cfg.MaxPayloadSizePerBatch,
		minPayloadSizePerBatch: cfg.MinPayloadSizePerBatch,
		maxChunkNumPerBatch:    cfg.MaxChunkNumPerBatch,
	}
}

func (p *BatchProposer) TryProposeBatch() {
	batchChunks, err := p.proposeBatchChunks()
	if err != nil {
		log.Error("proposeBatch failed", "err", err)
		return
	}
	if err := p.batchOrm.InsertBatch(p.ctx, batchChunks, p.chunkOrm, p.l2Block); err != nil {
		log.Error("InsertBatch failed", "err", err)
	}
}

func (p *BatchProposer) proposeBatchChunks() ([]*bridgeTypes.Chunk, error) {
	dbChunks, err := p.chunkOrm.GetUnbatchedChunks(p.ctx)
	if err != nil {
		return nil, err
	}

	if len(dbChunks) == 0 {
		return nil, errors.New("No Unbatched Chunks")
	}

	firstChunk := dbChunks[0]
	totalPayloadSize := firstChunk.TotalPayloadSize

	if totalPayloadSize > p.maxPayloadSizePerBatch {
		log.Warn("The first chunk exceeds the max payload size limit", "total payload size", totalPayloadSize, "max payload size limit", p.maxPayloadSizePerBatch)
		return p.convertToBridgeBlock(dbChunks[:1])
	}

	for i, chunk := range dbChunks[1:] {
		totalPayloadSize += chunk.TotalPayloadSize
		if totalPayloadSize > p.maxPayloadSizePerBatch {
			return p.convertToBridgeBlock(dbChunks[:i+1])
		}
	}

	if totalPayloadSize < p.minPayloadSizePerBatch {
		errMsg := fmt.Sprintf("The payload size of the batch is less than the minimum limit: %d", totalPayloadSize)
		return nil, errors.New(errMsg)
	}
	return p.convertToBridgeBlock(dbChunks)
}

func (p *BatchProposer) convertToBridgeBlock(dbChunks []*orm.Chunk) ([]*bridgeTypes.Chunk, error) {
	chunks := make([]*bridgeTypes.Chunk, len(dbChunks))
	for i, c := range dbChunks {
		wrappedBlocks, err := p.l2Block.RangeGetL2Blocks(p.ctx, c.StartBlockNumber, c.EndBlockNumber)
		if err != nil {
			log.Error("Failed to fetch wrapped blocks", "error", err)
			return nil, err
		}
		chunks[i].Blocks = wrappedBlocks
	}
	return chunks, nil
}
