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

type ChunkProposer struct {
	ctx context.Context
	db  *gorm.DB

	chunkOrm   *orm.Chunk
	l2BlockOrm *orm.L2Block

	maxGasPerChunk         uint64
	maxTxNumPerChunk       uint64
	maxPayloadSizePerChunk uint64
	minPayloadSizePerChunk uint64
}

func NewChunkProposer(ctx context.Context, cfg *config.ChunkProposerConfig, db *gorm.DB) *ChunkProposer {
	return &ChunkProposer{
		ctx:                    ctx,
		db:                     db,
		chunkOrm:               orm.NewChunk(db),
		l2BlockOrm:             orm.NewL2Block(db),
		maxGasPerChunk:         cfg.MaxGasPerChunk,
		maxTxNumPerChunk:       cfg.MaxTxNumPerChunk,
		maxPayloadSizePerChunk: cfg.MaxPayloadSizePerChunk,
		minPayloadSizePerChunk: cfg.MinPayloadSizePerChunk,
	}
}

func (p *ChunkProposer) TryProposeChunk() {
	proposedChunk, err := p.proposeChunk()
	if err != nil {
		log.Error("proposeChunk failed", "err", err)
		return
	}
	if err := p.chunkOrm.InsertChunk(p.ctx, proposedChunk, p.l2BlockOrm); err != nil {
		log.Error("InsertChunk failed", "err", err)
	}
}

func (p *ChunkProposer) proposeChunk() (*bridgeTypes.Chunk, error) {
	blocks, err := p.l2BlockOrm.GetUnchunkedBlocks()
	if err != nil {
		return nil, err
	}

	if len(blocks) == 0 {
		return nil, errors.New("No Unchunked Blocks")
	}

	approximatePayloadSize := func(block *bridgeTypes.WrappedBlock) (size uint64) {
		for _, tx := range block.Transactions {
			size += uint64(len(tx.Data))
		}
		return
	}

	firstBlock := blocks[0]
	totalGasUsed := firstBlock.Header.GasUsed
	totalTxNum := uint64(len(firstBlock.Transactions))
	totalPayloadSize := approximatePayloadSize(firstBlock)

	// Use the first block to propose a chunk if it exceeds any limits
	if totalGasUsed > p.maxGasPerChunk {
		log.Warn("The first block exceeds the max gas limit", "block number", firstBlock.Header.Number, "gas used", totalGasUsed, "max gas limit", p.maxGasPerChunk)
		return &bridgeTypes.Chunk{Blocks: blocks[:1]}, nil
	}
	if totalTxNum > p.maxTxNumPerChunk {
		log.Warn("The first block exceeds the max transaction number limit", "block number", firstBlock.Header.Number, "number of transactions", totalTxNum, "max transaction number limit", p.maxTxNumPerChunk)
		return &bridgeTypes.Chunk{Blocks: blocks[:1]}, nil
	}
	if totalPayloadSize > p.maxPayloadSizePerChunk {
		log.Warn("The first block exceeds the max calldata size limit", "block number", firstBlock.Header.Number, "calldata size", totalPayloadSize, "max calldata size limit", p.maxPayloadSizePerChunk)
		return &bridgeTypes.Chunk{Blocks: blocks[:1]}, nil
	}

	for i, block := range blocks[1:] {
		blockGasUsed := block.Header.GasUsed
		blockCalldataSize := approximatePayloadSize(block)
		blockTxNum := uint64(len(block.Transactions))
		if (totalGasUsed+blockGasUsed > p.maxGasPerChunk) || (totalTxNum+blockTxNum > p.maxTxNumPerChunk) || (totalPayloadSize+blockCalldataSize > p.maxPayloadSizePerChunk) {
			blocks = blocks[:i]
			break
		}
		totalGasUsed += blockGasUsed
		totalTxNum += blockTxNum
		totalPayloadSize += blockCalldataSize
	}

	if totalPayloadSize < p.minPayloadSizePerChunk {
		errMsg := fmt.Sprintf("The calldata size of the chunk is less than the minimum limit", "calldata size", totalPayloadSize)
		return nil, errors.New(errMsg)
	}
	return &bridgeTypes.Chunk{Blocks: blocks}, nil
}
