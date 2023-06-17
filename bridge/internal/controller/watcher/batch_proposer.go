package watcher

import (
	"context"
	"encoding/hex"
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

	maxChunkNumPerBatch             uint64
	maxL1CommitGasPerBatch          uint64
	maxL1CommitCalldataSizePerBatch uint64
	minChunkNumPerBatch             uint64
	batchTimeoutSec                 uint64
}

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

func (p *BatchProposer) TryProposeBatch() {
	batchChunks, err := p.proposeBatchChunks()
	if err != nil {
		log.Error("proposeBatch failed", "err", err)
		return
	}
	if err := p.updateBatchInfoInDB(batchChunks); err != nil {
		log.Error("update batch info in db failed", "err", err)
	}
}

func (p *BatchProposer) updateBatchInfoInDB(chunks []*bridgeTypes.Chunk) error {
	numChunks := len(chunks)
	if numChunks <= 0 {
		return nil
	}
	startDBChunk, err := p.chunkOrm.GetChunkByStartBlockIndex(p.ctx, chunks[0].Blocks[0].Header.Number.Uint64())
	if err != nil {
		return err
	}
	startChunkIndex := startDBChunk.Index

	endDBChunk, err := p.chunkOrm.GetChunkByStartBlockIndex(p.ctx, chunks[numChunks-1].Blocks[0].Header.Number.Uint64())
	if err != nil {
		return err
	}
	endChunkIndex := endDBChunk.Index

	startChunkHashBytes, err := chunks[0].Hash(startDBChunk.TotalL1MessagePoppedBefore)
	if err != nil {
		return err
	}
	startChunkHash := hex.EncodeToString(startChunkHashBytes)

	endChunkHashBytes, err := chunks[numChunks-1].Hash(endDBChunk.TotalL1MessagePoppedBefore)
	if err != nil {
		return err
	}
	endChunkHash := hex.EncodeToString(endChunkHashBytes)

	err = p.db.Transaction(func(dbTX *gorm.DB) error {
		batchHash, err := p.batchOrm.InsertBatch(p.ctx, startChunkIndex, endChunkIndex, startChunkHash, endChunkHash, chunks, dbTX)
		if err != nil {
			return err
		}
		err = p.chunkOrm.UpdateBatchHashInClosedRange(p.ctx, startChunkIndex, endChunkIndex, batchHash, dbTX)
		if err != nil {
			return err
		}
		return nil
	})
	return err
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

	if totalPayloadSize > p.maxL1CommitCalldataSizePerBatch {
		log.Warn("The first chunk exceeds the max payload size limit",
			"total payload size", totalPayloadSize,
			"max payload size limit", p.maxL1CommitCalldataSizePerBatch,
		)
		return p.dbChunksToBridgeChunks(dbChunks[:1])
	}

	for i, chunk := range dbChunks[1:] {
		totalPayloadSize += chunk.TotalPayloadSize
		if totalPayloadSize > p.maxL1CommitCalldataSizePerBatch {
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

	if !hasChunkTimeout && uint64(len(dbChunks)) < p.minChunkNumPerBatch {
		log.Warn("The payload size of the batch is less than the minimum limit",
			"chunk num", len(dbChunks), "minChunkNumPerBatch", p.minChunkNumPerBatch,
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
