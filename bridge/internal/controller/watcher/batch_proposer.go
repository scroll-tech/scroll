package watcher

import (
	"context"

	"github.com/scroll-tech/go-ethereum/log"
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

func (p *BatchProposer) TryProposeBatch() {
	dbChunks, err := p.chunkOrm.GetUnbatchedChunks(p.ctx)
	if err != nil {
		log.Error("Failed to get unbatched chunks", "error", err)
		return
	}

	if len(dbChunks) == 0 {
		log.Warn("No unbatched chunks found")
		return
	}

	chunks := make([]*bridgeTypes.Chunk, 0, len(dbChunks))
	for _, chunk := range dbChunks {
		wrappedBlocks, err := p.l2BlockOrm.RangeGetL2WrappedBlocks(chunk.StartBlockNumber, chunk.EndBlockNumber)
		if err != nil {
			log.Error("Failed to get wrapped blocks for chunk", "error", err)
			return
		}
		chunks = append(chunks, &bridgeTypes.Chunk{Blocks: wrappedBlocks})
	}

	if err := p.batchOrm.InsertBatch(p.ctx, chunks, p.chunkOrm, p.l2BlockOrm); err != nil {
		log.Error("Failed to insert chunks into batch", "error", err)
	}
}
