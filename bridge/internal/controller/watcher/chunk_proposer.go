package watcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/core/types"
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
	maxL2TxNumPerChunk     uint64
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
		maxL2TxNumPerChunk:     cfg.MaxL2TxNumPerChunk,
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
		// TODO: implement an exact calculation.
		for _, tx := range block.Transactions {
			size += uint64(len(tx.Data))
		}
		return
	}

	firstBlock := blocks[0]
	totalGasUsed := firstBlock.Header.GasUsed
	totalL2TxNum := getL2TxsNum(firstBlock.Transactions)
	totalPayloadSize := approximatePayloadSize(firstBlock)

	// If the total number of L2 transactions in a single block exceeds
	// the maximum limit set per chunk, the chunk proposer will get stuck
	// and keep returning an error.
	// In such a scenario, manual intervention is needed to resolve the issue.
	// This should not happen in practice because l2geth enforces the same limit.
	if totalL2TxNum > p.maxL2TxNumPerChunk {
		errMsg := fmt.Sprintln("The first block exceeds the max transaction number limit", "block number", firstBlock.Header.Number, "number of transactions", totalTxNum, "max transaction number limit", p.maxL2TxNumPerChunk)
		log.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// Use the first block to propose a chunk if it exceeds any limits
	if totalGasUsed > p.maxGasPerChunk {
		log.Warn("The first block exceeds the max gas limit", "block number", firstBlock.Header.Number, "gas used", totalGasUsed, "max gas limit", p.maxGasPerChunk)
		return &bridgeTypes.Chunk{Blocks: blocks[:1]}, nil
	}
	if totalPayloadSize > p.maxPayloadSizePerChunk {
		log.Warn("The first block exceeds the max calldata size limit", "block number", firstBlock.Header.Number, "calldata size", totalPayloadSize, "max calldata size limit", p.maxPayloadSizePerChunk)
		return &bridgeTypes.Chunk{Blocks: blocks[:1]}, nil
	}

	for i, block := range blocks[1:] {
		totalGasUsed += block.Header.GasUsed
		totalL2TxNum += getL2TxsNum(block.Transactions)
		totalPayloadSize += approximatePayloadSize(block)
		if (totalGasUsed > p.maxGasPerChunk) || (totalL2TxNum > p.maxL2TxNumPerChunk) || (totalPayloadSize > p.maxPayloadSizePerChunk) {
			blocks = blocks[:i+1]
			break
		}
	}

	if totalPayloadSize < p.minPayloadSizePerChunk {
		log.Warn("The calldata size of the chunk is less than the minimum limit",
			"totalPayloadSize", totalPayloadSize,
			"minPayloadSizePerChunk", p.minPayloadSizePerChunk,
		)
		return nil, nil
	}
	return &bridgeTypes.Chunk{Blocks: blocks}, nil
}

func getL2TxsNum(txs []*types.TransactionData) (count uint64) {
	for _, tx := range txs {
		// TODO(colinlyguo): replace with L1MessageTxType after upgrading github.com/scroll-tech/go-ethereum version in go.mod.
		if tx.Type == types.LegacyTxType || tx.Type == types.AccessListTxType || tx.Type == types.DynamicFeeTxType {
			count += 1
		}
	}
	return
}
