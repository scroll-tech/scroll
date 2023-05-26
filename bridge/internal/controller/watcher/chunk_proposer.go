package watcher

import (
	"context"

	"gorm.io/gorm"

	"scroll-tech/bridge/internal/controller/relayer"
	"scroll-tech/bridge/internal/orm"
)

type ChunkProposer struct {
	ctx context.Context
	db  *gorm.DB

	relayer *relayer.Layer2Relayer

	blockContextOrm *orm.BlockContext
	blockTraceOrm   *orm.BlockTrace
}

func NewChunkProposer(ctx context.Context, relayer *relayer.Layer2Relayer, db *gorm.DB) *ChunkProposer {
	return &ChunkProposer{
		ctx:             ctx,
		db:              db,
		blockContextOrm: orm.NewBlockContext(db),
		blockTraceOrm:   orm.NewBlockTrace(db),
		relayer:         relayer,
	}
}
