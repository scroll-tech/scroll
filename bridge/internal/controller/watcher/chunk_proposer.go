package watcher

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
)

// ChunkProposer proposes chunks based on available unchunked blocks.
type ChunkProposer struct {
	ctx context.Context
	db  *gorm.DB

	chunkOrm   *orm.Chunk
	l2BlockOrm *orm.L2Block

	maxL2TxGasPerChunk              uint64
	maxL2TxNumPerChunk              uint64
	maxL1CommitGasPerChunk          uint64
	maxL1CommitCalldataSizePerChunk uint64
	minL1CommitCalldataSizePerChunk uint64
	chunkTimeoutSec                 uint64
}

// NewChunkProposer creates a new ChunkProposer instance.
func NewChunkProposer(ctx context.Context, cfg *config.ChunkProposerConfig, db *gorm.DB) *ChunkProposer {
	return &ChunkProposer{
		ctx:                             ctx,
		db:                              db,
		chunkOrm:                        orm.NewChunk(db),
		l2BlockOrm:                      orm.NewL2Block(db),
		maxL2TxGasPerChunk:              cfg.MaxL2TxGasPerChunk,
		maxL2TxNumPerChunk:              cfg.MaxL2TxNumPerChunk,
		maxL1CommitGasPerChunk:          cfg.MaxL1CommitGasPerChunk,
		maxL1CommitCalldataSizePerChunk: cfg.MaxL1CommitCalldataSizePerChunk,
		minL1CommitCalldataSizePerChunk: cfg.MinL1CommitCalldataSizePerChunk,
		chunkTimeoutSec:                 cfg.ChunkTimeoutSec,
	}
}

// TryProposeChunk tries to propose a new chunk.
func (p *ChunkProposer) TryProposeChunk() {
	proposedChunk, err := p.proposeChunk()
	if err != nil {
		log.Error("propose new chunk failed", "err", err)
		return
	}
	if err := p.updateChunkInfoInDB(proposedChunk); err != nil {
		log.Error("update chunk info in orm failed", "err", err)
	}
}

func (p *ChunkProposer) updateChunkInfoInDB(chunk *bridgeTypes.Chunk) error {
	err := p.db.Transaction(func(dbTX *gorm.DB) error {
		chunkHash, err := p.chunkOrm.InsertChunk(p.ctx, chunk)
		if err != nil {
			return err
		}
		startBlockNum := chunk.Blocks[0].Header.Number.Uint64()
		endBlockNum := startBlockNum + uint64(len(chunk.Blocks))
		if err := p.l2BlockOrm.UpdateChunkHashInClosedRange(startBlockNum, endBlockNum, chunkHash); err != nil {
			log.Error("failed to update chunk_hash for l2_blocks", "chunk hash", chunkHash, "start block", 0, "end block", 0, "err", err)
			return err
		}
		return nil
	})
	return err
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
		errMsg := fmt.Sprintln("The first block exceeds the max transaction number limit", "block number", firstBlock.Header.Number, "number of transactions", totalL2TxNum, "max transaction number limit", p.maxL2TxNumPerChunk)
		log.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// Use the first block to propose a chunk if it exceeds any limits
	if totalGasUsed > p.maxL2TxGasPerChunk {
		log.Warn("The first block exceeds the max gas limit", "block number", firstBlock.Header.Number, "gas used", totalGasUsed, "max gas limit", p.maxL2TxGasPerChunk)
		return &bridgeTypes.Chunk{Blocks: blocks[:1]}, nil
	}
	if totalPayloadSize > p.maxL1CommitCalldataSizePerChunk {
		log.Warn("The first block exceeds the max calldata size limit", "block number", firstBlock.Header.Number, "calldata size", totalPayloadSize, "max calldata size limit", p.maxL1CommitCalldataSizePerChunk)
		return &bridgeTypes.Chunk{Blocks: blocks[:1]}, nil
	}

	for i, block := range blocks[1:] {
		totalGasUsed += block.Header.GasUsed
		totalL2TxNum += getL2TxsNum(block.Transactions)
		totalPayloadSize += approximatePayloadSize(block)
		if (totalGasUsed > p.maxL2TxGasPerChunk) || (totalL2TxNum > p.maxL2TxNumPerChunk) || (totalPayloadSize > p.maxL1CommitCalldataSizePerChunk) {
			blocks = blocks[:i+1]
			break
		}
	}

	var hasBlockTimeout bool
	currentTimeSec := uint64(time.Now().Unix())
	if blocks[0].Header.Time+p.chunkTimeoutSec > currentTimeSec {
		log.Warn("first block timeout", "block number", blocks[0].Header.Number, "block timestamp", blocks[0].Header.Time, "chunk time limit", currentTimeSec)
		hasBlockTimeout = true
	}

	if !hasBlockTimeout && totalPayloadSize < p.minL1CommitCalldataSizePerChunk {
		log.Warn("The calldata size of the chunk is less than the minimum limit",
			"totalPayloadSize", totalPayloadSize,
			"minL1CommitCalldataSizePerChunk", p.minL1CommitCalldataSizePerChunk,
		)
		return nil, nil
	}
	return &bridgeTypes.Chunk{Blocks: blocks}, nil
}

func getL2TxsNum(txs []*types.TransactionData) (count uint64) {
	for _, tx := range txs {
		// TODO(colinlyguo): replace with L1MessageTxType after upgrading github.com/scroll-tech/go-ethereum version in go.mod.
		if tx.Type == types.LegacyTxType || tx.Type == types.AccessListTxType || tx.Type == types.DynamicFeeTxType {
			count++
		}
	}
	return
}
