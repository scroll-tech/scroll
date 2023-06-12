package watcher

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
)

// BatchProposer sends batches commit transactions to relayer.
type BatchProposer struct {
	ctx context.Context
	db  *gorm.DB

	batchOrm   *orm.Batch
	l2BlockOrm *orm.L2Block
	chunkOrm   *orm.Chunk
}

// NewBatchProposer will return a new instance of BatchProposer.
func NewBatchProposer(ctx context.Context, db *gorm.DB) *BatchProposer {
	p := &BatchProposer{
		ctx:        ctx,
		db:         db,
		batchOrm:   orm.NewBatch(db),
		l2BlockOrm: orm.NewL2Block(db),
		chunkOrm:   orm.NewChunk(db),
	}

	return p
}

func (p *BatchProposer) TryProposeBatch() error {
	dbChunks, err := p.chunkOrm.GetUnbatchedChunks(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to get unbatched chunks: %w", err)
	}

	chunks := make([]*bridgeTypes.Chunk, len(dbChunks))
	for i, chunk := range dbChunks {
		wrappedBlocks, err := p.l2BlockOrm.GetL2WrappedBlocksRange(chunk.StartBlockNumber, chunk.EndBlockNumber)
		if err != nil {
			return fmt.Errorf("failed to get wrapped blocks for chunk: %w", err)
		}

		chunks[i] = &bridgeTypes.Chunk{
			Blocks: wrappedBlocks,
		}
	}

	if err := p.batchOrm.InsertBatch(p.ctx, chunks, p.chunkOrm, p.l2BlockOrm); err != nil {
		return fmt.Errorf("failed to insert chunks into batch: %w", err)
	}

	return nil
}
