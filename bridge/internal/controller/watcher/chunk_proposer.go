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

func (p *ChunkProposer) TryProposeChunk() {
	// TODO: refine strategy
	wrappedBlocks, err := p.l2BlockOrm.GetUnchunkedBlocks()
	if err != nil {
		log.Error("GetUnchunkedBlocks", "err", err)
		return
	}

	if err := p.chunkOrm.InsertChunk(p.ctx, &types.Chunk{Blocks: wrappedBlocks}, p.l2BlockOrm); err != nil {
		log.Error("InsertChunk failed", "err", err)
		return
	}

	return
}
