package watcher

import (
	"context"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge/internal/orm"
	"scroll-tech/bridge/internal/types"
)

type ChunkProposer struct {
	ctx context.Context
	db  *gorm.DB

	chunkOrm   *orm.Chunk
	l2BlockOrm *orm.L2Block
}

func NewChunkProposer(ctx context.Context, db *gorm.DB) *ChunkProposer {
	return &ChunkProposer{
		ctx:        ctx,
		db:         db,
		chunkOrm:   orm.NewChunk(db),
		l2BlockOrm: orm.NewL2Block(db),
	}
}

func (c *ChunkProposer) TryProposeChunk() {
	// TODO: refine strategy
	// Fetch the latest 10 blocks
	var fields map[string]interface{}
	orderByList := []string{"number DESC"}
	limit := 10
	wrappedBlocks, err := c.l2BlockOrm.GetL2WrappedBlocks(fields, orderByList, limit)
	if err != nil {
		log.Error("GetL2WrappedBlocks", "err", err)
		return
	}

	if len(wrappedBlocks) != limit {
		log.Info("Not enough block", "num", len(wrappedBlocks))
		return
	}

	if err := c.chunkOrm.InsertChunk(c.ctx, &types.Chunk{
		Blocks: wrappedBlocks,
	}); err != nil {
		log.Error("InsertChunk failed", "err", err)
		return
	}

	return
}
