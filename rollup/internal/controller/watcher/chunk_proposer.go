package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/forks"
	"scroll-tech/common/types/encoding"
	"scroll-tech/common/types/encoding/codecv0"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
)

// ChunkProposer proposes chunks based on available unchunked blocks.
type ChunkProposer struct {
	ctx context.Context
	db  *gorm.DB

	chunkOrm   *orm.Chunk
	l2BlockOrm *orm.L2Block

	maxBlockNumPerChunk             uint64
	maxTxNumPerChunk                uint64
	maxL1CommitGasPerChunk          uint64
	maxL1CommitCalldataSizePerChunk uint64
	maxRowConsumptionPerChunk       uint64
	chunkTimeoutSec                 uint64
	gasCostIncreaseMultiplier       float64
	forkHeights                     []uint64

	chunkProposerCircleTotal           prometheus.Counter
	proposeChunkFailureTotal           prometheus.Counter
	proposeChunkUpdateInfoTotal        prometheus.Counter
	proposeChunkUpdateInfoFailureTotal prometheus.Counter
	chunkTxNum                         prometheus.Gauge
	chunkEstimateL1CommitGas           prometheus.Gauge
	totalL1CommitCalldataSize          prometheus.Gauge
	maxTxConsumption                   prometheus.Gauge
	chunkBlocksNum                     prometheus.Gauge
	chunkFirstBlockTimeoutReached      prometheus.Counter
	chunkBlocksProposeNotEnoughTotal   prometheus.Counter
}

// NewChunkProposer creates a new ChunkProposer instance.
func NewChunkProposer(ctx context.Context, cfg *config.ChunkProposerConfig, chainCfg *params.ChainConfig, db *gorm.DB, reg prometheus.Registerer) *ChunkProposer {
	forkHeights, _ := forks.CollectSortedForkHeights(chainCfg)
	log.Debug("new chunk proposer",
		"maxTxNumPerChunk", cfg.MaxTxNumPerChunk,
		"maxL1CommitGasPerChunk", cfg.MaxL1CommitGasPerChunk,
		"maxL1CommitCalldataSizePerChunk", cfg.MaxL1CommitCalldataSizePerChunk,
		"maxRowConsumptionPerChunk", cfg.MaxRowConsumptionPerChunk,
		"chunkTimeoutSec", cfg.ChunkTimeoutSec,
		"gasCostIncreaseMultiplier", cfg.GasCostIncreaseMultiplier,
		"forkHeights", forkHeights)

	return &ChunkProposer{
		ctx:                             ctx,
		db:                              db,
		chunkOrm:                        orm.NewChunk(db),
		l2BlockOrm:                      orm.NewL2Block(db),
		maxBlockNumPerChunk:             cfg.MaxBlockNumPerChunk,
		maxTxNumPerChunk:                cfg.MaxTxNumPerChunk,
		maxL1CommitGasPerChunk:          cfg.MaxL1CommitGasPerChunk,
		maxL1CommitCalldataSizePerChunk: cfg.MaxL1CommitCalldataSizePerChunk,
		maxRowConsumptionPerChunk:       cfg.MaxRowConsumptionPerChunk,
		chunkTimeoutSec:                 cfg.ChunkTimeoutSec,
		gasCostIncreaseMultiplier:       cfg.GasCostIncreaseMultiplier,
		forkHeights:                     forkHeights,

		chunkProposerCircleTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_chunk_circle_total",
			Help: "Total number of propose chunk total.",
		}),
		proposeChunkFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_chunk_failure_circle_total",
			Help: "Total number of propose chunk failure total.",
		}),
		proposeChunkUpdateInfoTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_chunk_update_info_total",
			Help: "Total number of propose chunk update info total.",
		}),
		proposeChunkUpdateInfoFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_chunk_update_info_failure_total",
			Help: "Total number of propose chunk update info failure total.",
		}),
		chunkTxNum: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_chunk_tx_num",
			Help: "The chunk tx num",
		}),
		chunkEstimateL1CommitGas: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_chunk_estimate_l1_commit_gas",
			Help: "The chunk estimate l1 commit gas",
		}),
		totalL1CommitCalldataSize: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_chunk_total_l1_commit_call_data_size",
			Help: "The total l1 commit call data size",
		}),
		maxTxConsumption: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_chunk_max_tx_consumption",
			Help: "The max tx consumption",
		}),
		chunkBlocksNum: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_chunk_chunk_block_number",
			Help: "The number of blocks in the chunk",
		}),
		chunkFirstBlockTimeoutReached: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_chunk_first_block_timeout_reached_total",
			Help: "Total times of chunk's first block timeout reached",
		}),
		chunkBlocksProposeNotEnoughTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_chunk_blocks_propose_not_enough_total",
			Help: "Total number of chunk block propose not enough",
		}),
	}
}

// TryProposeChunk tries to propose a new chunk.
func (p *ChunkProposer) TryProposeChunk() {
	p.chunkProposerCircleTotal.Inc()
	proposedChunk, err := p.proposeChunk()
	if err != nil {
		p.proposeChunkFailureTotal.Inc()
		log.Error("propose new chunk failed", "err", err)
		return
	}

	if err := p.updateChunkInfoInDB(proposedChunk); err != nil {
		p.proposeChunkUpdateInfoFailureTotal.Inc()
		log.Error("update chunk info in orm failed", "err", err)
	}
}

func (p *ChunkProposer) updateChunkInfoInDB(chunk *encoding.Chunk) error {
	if chunk == nil {
		return nil
	}

	p.proposeChunkUpdateInfoTotal.Inc()
	err := p.db.Transaction(func(dbTX *gorm.DB) error {
		dbChunk, err := p.chunkOrm.InsertChunk(p.ctx, chunk, dbTX)
		if err != nil {
			log.Warn("ChunkProposer.InsertChunk failed", "err", err)
			return err
		}
		if err := p.l2BlockOrm.UpdateChunkHashInRange(p.ctx, dbChunk.StartBlockNumber, dbChunk.EndBlockNumber, dbChunk.Hash, dbTX); err != nil {
			log.Error("failed to update chunk_hash for l2_blocks", "chunk hash", dbChunk.Hash, "start block", dbChunk.StartBlockNumber, "end block", dbChunk.EndBlockNumber, "err", err)
			return err
		}
		return nil
	})
	return err
}

func (p *ChunkProposer) proposeChunk() (*encoding.Chunk, error) {
	unchunkedBlockHeight, err := p.chunkOrm.GetUnchunkedBlockHeight(p.ctx)
	if err != nil {
		return nil, err
	}

	maxBlocksThisChunk := p.maxBlockNumPerChunk
	blocksUntilFork := forks.BlocksUntilFork(unchunkedBlockHeight, p.forkHeights)
	if blocksUntilFork != 0 && blocksUntilFork < maxBlocksThisChunk {
		maxBlocksThisChunk = blocksUntilFork
	}

	// select at most maxBlocksThisChunk blocks
	blocks, err := p.l2BlockOrm.GetL2BlocksGEHeight(p.ctx, unchunkedBlockHeight, int(maxBlocksThisChunk))
	if err != nil {
		return nil, err
	}

	if len(blocks) == 0 {
		return nil, nil
	}

	var chunk encoding.Chunk
	for i, block := range blocks {
		chunk.Blocks = append(chunk.Blocks, block)

		crcMax, err := chunk.CrcMax()
		if err != nil {
			return nil, fmt.Errorf("failed to get crc max: %w", err)
		}

		totalTxNum := chunk.NumTransactions()

		totalL1CommitCalldataSize, err := codecv0.EstimateChunkL1CommitCalldataSize(&chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate chunk L1 commit calldata size: %w", err)
		}

		totalL1CommitGas, err := codecv0.EstimateChunkL1CommitGas(&chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate chunk L1 commit gas: %w", err)
		}

		totalOverEstimateL1CommitGas := uint64(p.gasCostIncreaseMultiplier * float64(totalL1CommitGas))

		if totalTxNum > p.maxTxNumPerChunk ||
			totalL1CommitCalldataSize > p.maxL1CommitCalldataSizePerChunk ||
			totalOverEstimateL1CommitGas > p.maxL1CommitGasPerChunk ||
			crcMax > p.maxRowConsumptionPerChunk {
			// Check if the first block breaks hard limits.
			// If so, it indicates there are bugs in sequencer, manual fix is needed.
			if i == 0 {
				if totalTxNum > p.maxTxNumPerChunk {
					return nil, fmt.Errorf(
						"the first block exceeds l2 tx number limit; block number: %v, number of transactions: %v, max transaction number limit: %v",
						block.Header.Number,
						totalTxNum,
						p.maxTxNumPerChunk,
					)
				}

				if totalOverEstimateL1CommitGas > p.maxL1CommitGasPerChunk {
					return nil, fmt.Errorf(
						"the first block exceeds l1 commit gas limit; block number: %v, commit gas: %v, max commit gas limit: %v",
						block.Header.Number,
						totalL1CommitGas,
						p.maxL1CommitGasPerChunk,
					)
				}

				if totalL1CommitCalldataSize > p.maxL1CommitCalldataSizePerChunk {
					return nil, fmt.Errorf(
						"the first block exceeds l1 commit calldata size limit; block number: %v, calldata size: %v, max calldata size limit: %v",
						block.Header.Number,
						totalL1CommitCalldataSize,
						p.maxL1CommitCalldataSizePerChunk,
					)
				}

				if crcMax > p.maxRowConsumptionPerChunk {
					return nil, fmt.Errorf(
						"the first block exceeds row consumption limit; block number: %v, crc max: %v, limit: %v",
						block.Header.Number,
						crcMax,
						p.maxRowConsumptionPerChunk,
					)
				}
			}

			log.Debug("breaking limit condition in chunking",
				"totalTxNum", totalTxNum,
				"maxTxNumPerChunk", p.maxTxNumPerChunk,
				"currentL1CommitCalldataSize", totalL1CommitCalldataSize,
				"maxL1CommitCalldataSizePerChunk", p.maxL1CommitCalldataSizePerChunk,
				"currentOverEstimateL1CommitGas", totalOverEstimateL1CommitGas,
				"maxL1CommitGasPerChunk", p.maxL1CommitGasPerChunk,
				"chunkRowConsumptionMax", crcMax,
				"p.maxRowConsumptionPerChunk", p.maxRowConsumptionPerChunk)

			chunk.Blocks = chunk.Blocks[:len(chunk.Blocks)-1]

			crcMax, err := chunk.CrcMax()
			if err != nil {
				return nil, fmt.Errorf("failed to get crc max: %w", err)
			}

			totalL1CommitCalldataSize, err := codecv0.EstimateChunkL1CommitCalldataSize(&chunk)
			if err != nil {
				return nil, fmt.Errorf("failed to estimate chunk L1 commit calldata size: %w", err)
			}

			totalL1CommitGas, err := codecv0.EstimateChunkL1CommitGas(&chunk)
			if err != nil {
				return nil, fmt.Errorf("failed to estimate chunk L1 commit gas: %w", err)
			}

			p.chunkTxNum.Set(float64(chunk.NumTransactions()))
			p.totalL1CommitCalldataSize.Set(float64(totalL1CommitCalldataSize))
			p.chunkEstimateL1CommitGas.Set(float64(totalL1CommitGas))
			p.maxTxConsumption.Set(float64(crcMax))
			p.chunkBlocksNum.Set(float64(len(chunk.Blocks)))
			return &chunk, nil
		}
	}

	currentTimeSec := uint64(time.Now().Unix())
	if chunk.Blocks[0].Header.Time+p.chunkTimeoutSec < currentTimeSec ||
		uint64(len(chunk.Blocks)) == maxBlocksThisChunk {
		if chunk.Blocks[0].Header.Time+p.chunkTimeoutSec < currentTimeSec {
			log.Warn("first block timeout",
				"block number", chunk.Blocks[0].Header.Number,
				"block timestamp", chunk.Blocks[0].Header.Time,
				"current time", currentTimeSec,
			)
		} else {
			log.Info("reached maximum number of blocks in chunk",
				"start block number", chunk.Blocks[0].Header.Number,
				"block count", len(chunk.Blocks),
			)
		}

		crcMax, err := chunk.CrcMax()
		if err != nil {
			return nil, fmt.Errorf("failed to get crc max: %w", err)
		}

		totalL1CommitCalldataSize, err := codecv0.EstimateChunkL1CommitCalldataSize(&chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate chunk L1 commit calldata size: %w", err)
		}

		totalL1CommitGas, err := codecv0.EstimateChunkL1CommitGas(&chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate chunk L1 commit gas: %w", err)
		}

		p.chunkFirstBlockTimeoutReached.Inc()
		p.chunkTxNum.Set(float64(chunk.NumTransactions()))
		p.totalL1CommitCalldataSize.Set(float64(totalL1CommitCalldataSize))
		p.chunkEstimateL1CommitGas.Set(float64(totalL1CommitGas))
		p.maxTxConsumption.Set(float64(crcMax))
		p.chunkBlocksNum.Set(float64(len(chunk.Blocks)))
		return &chunk, nil
	}

	log.Debug("pending blocks do not reach one of the constraints or contain a timeout block")
	p.chunkBlocksProposeNotEnoughTotal.Inc()
	return nil, nil
}
