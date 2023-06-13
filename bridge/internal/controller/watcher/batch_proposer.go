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

	maxGasPerBatch         uint64
	maxTxNumPerBatch       uint64
	maxPayloadSizePerBatch uint64
	minPayloadSizePerBatch uint64
}

func NewBatchProposer(ctx context.Context, cfg *config.BatchProposerConfig, db *gorm.DB) *BatchProposer {
	return &BatchProposer{
		ctx:                    ctx,
		db:                     db,
		batchOrm:               orm.NewBatch(db),
		chunkOrm:               orm.NewChunk(db),
		l2Block:                orm.NewL2Block(db),
		maxGasPerBatch:         cfg.MaxGasPerBatch,
		maxTxNumPerBatch:       cfg.MaxTxNumPerBatch,
		maxPayloadSizePerBatch: cfg.MaxPayloadSizePerBatch,
		minPayloadSizePerBatch: cfg.MinPayloadSizePerBatch,
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
	totalGasUsed := firstChunk.TotalGasUsed
	totalTxNum := firstChunk.TotalTxNum
	totalPayloadSize := firstChunk.TotalPayloadSize

	if totalGasUsed > p.maxGasPerBatch {
		log.Warn("The first chunk exceeds the max gas limit", "total gas used", totalGasUsed, "max gas limit", p.maxGasPerBatch)
		return convertToBridgeBlock(dbChunks[:1])
	}

	if totalTxNum > p.maxTxNumPerBatch {
		log.Warn("The first chunk exceeds the max transaction number limit", "total transaction number", totalTxNum, "max transaction number limit", p.maxTxNumPerBatch)
		return convertToBridgeBlock(dbChunks[:1])
	}

	if totalPayloadSize > p.maxPayloadSizePerBatch {
		log.Warn("The first chunk exceeds the max payload size limit", "total payload size", totalPayloadSize, "max payload size limit", p.maxPayloadSizePerBatch)
		return convertToBridgeBlock(dbChunks[:1])
	}

	for i, chunk := range dbChunks[1:] {
		if (totalGasUsed+chunk.TotalGasUsed > p.maxGasPerBatch) || (totalTxNum+chunk.TotalTxNum > p.maxTxNumPerBatch) || (totalPayloadSize+chunk.TotalPayloadSize > p.maxPayloadSizePerBatch) {
			dbChunks = dbChunks[:i]
			break
		}
		totalGasUsed += chunk.TotalGasUsed
		totalTxNum += chunk.TotalTxNum
		totalPayloadSize += chunk.TotalPayloadSize
	}

	if totalPayloadSize < p.minPayloadSizePerBatch {
		errMsg := fmt.Sprintf("The payload size of the batch is less than the minimum limit", "payload size", totalPayloadSize)
		return nil, errors.New(errMsg)
	}
	return convertToBridgeBlock(dbChunks)
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
