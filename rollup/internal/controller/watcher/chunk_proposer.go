package watcher

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/forks"
	"scroll-tech/common/types/encoding"
	"scroll-tech/common/types/encoding/codecv0"
	"scroll-tech/common/types/encoding/codecv1"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
)

var maxBlobSize = uint64(131072)

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
	banachForkHeight                uint64

	chunkProposerCircleTotal           prometheus.Counter
	proposeChunkFailureTotal           prometheus.Counter
	proposeChunkUpdateInfoTotal        prometheus.Counter
	proposeChunkUpdateInfoFailureTotal prometheus.Counter
	chunkTxNum                         prometheus.Gauge
	chunkEstimateL1CommitGas           prometheus.Gauge
	totalL1CommitCalldataSize          prometheus.Gauge
	totalL1CommitBlobSize              prometheus.Gauge
	maxTxConsumption                   prometheus.Gauge
	chunkBlocksNum                     prometheus.Gauge
	chunkFirstBlockTimeoutReached      prometheus.Counter
	chunkBlocksProposeNotEnoughTotal   prometheus.Counter
}

type chunkMetrics struct {
	// common metrics
	numBlocks           uint64
	txNum               uint64
	crcMax              uint64
	firstBlockTimestamp uint64

	// codecv0 metrics, default 0 for codecv1
	l1CommitCalldataSize uint64
	l1CommitGas          uint64

	// codecv1 metrics, default 0 for codecv0
	l1CommitBlobSize uint64
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

	p := &ChunkProposer{
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
		totalL1CommitBlobSize: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_chunk_total_l1_commit_blob_size",
			Help: "The total l1 commit blob size",
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

	// If BanachBlock is not set in chain's genesis config, banachForkHeight is inf,
	// which means chunk proposer uses the codecv0 version by default.
	// TODO: Must change it to real fork name.
	if chainCfg.BanachBlock != nil {
		p.banachForkHeight = chainCfg.BanachBlock.Uint64()
	} else {
		p.banachForkHeight = math.MaxUint64
	}
	return p
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

		metrics, calcErr := p.calculateChunkMetrics(&chunk)
		if calcErr != nil {
			return nil, fmt.Errorf("failed to calculate chunk metrics: %w", calcErr)
		}

		overEstimatedL1CommitGas := uint64(p.gasCostIncreaseMultiplier * float64(metrics.l1CommitGas))
		if metrics.txNum > p.maxTxNumPerChunk ||
			metrics.l1CommitCalldataSize > p.maxL1CommitCalldataSizePerChunk ||
			overEstimatedL1CommitGas > p.maxL1CommitGasPerChunk ||
			metrics.crcMax > p.maxRowConsumptionPerChunk ||
			metrics.l1CommitBlobSize > maxBlobSize {
			if i == 0 {
				// The first block exceeds hard limits, which indicates a bug in the sequencer, manual fix is needed.
				return nil, fmt.Errorf(
					"the first block exceeds limits; block number: %v, limits: %+v, maxTxNum: %v, maxL1CommitCalldataSize: %v, maxL1CommitGas: %v, maxRowConsumption: %v, maxBlobSize: %v",
					block.Header.Number, metrics, p.maxTxNumPerChunk, p.maxL1CommitCalldataSizePerChunk, p.maxL1CommitGasPerChunk, p.maxRowConsumptionPerChunk, maxBlobSize)
			}

			log.Debug("breaking limit condition in chunking",
				"txNum", metrics.txNum,
				"maxTxNum", p.maxTxNumPerChunk,
				"l1CommitCalldataSize", metrics.l1CommitCalldataSize,
				"maxL1CommitCalldataSize", p.maxL1CommitCalldataSizePerChunk,
				"overEstimatedL1CommitGas", overEstimatedL1CommitGas,
				"maxL1CommitGas", p.maxL1CommitGasPerChunk,
				"rowConsumption", metrics.crcMax,
				"maxRowConsumption", p.maxRowConsumptionPerChunk,
				"maxBlobSize", maxBlobSize)

			chunk.Blocks = chunk.Blocks[:len(chunk.Blocks)-1]

			metrics, calcErr := p.calculateChunkMetrics(&chunk)
			if calcErr != nil {
				return nil, fmt.Errorf("failed to calculate chunk metrics: %w", calcErr)
			}
			p.recordChunkMetrics(metrics)
			return &chunk, nil
		}
	}

	metrics, calcErr := p.calculateChunkMetrics(&chunk)
	if calcErr != nil {
		return nil, fmt.Errorf("failed to calculate chunk metrics: %w", calcErr)
	}
	currentTimeSec := uint64(time.Now().Unix())
	if metrics.firstBlockTimestamp+p.chunkTimeoutSec < currentTimeSec || metrics.numBlocks == maxBlocksThisChunk {
		log.Info("reached maximum number of blocks in chunk or first block timeout",
			"start block number", chunk.Blocks[0].Header.Number,
			"block count", len(chunk.Blocks),
			"block number", chunk.Blocks[0].Header.Number,
			"block timestamp", metrics.firstBlockTimestamp,
			"current time", currentTimeSec)

		p.chunkFirstBlockTimeoutReached.Inc()
		p.recordChunkMetrics(metrics)
		return &chunk, nil
	}

	log.Debug("pending blocks do not reach one of the constraints or contain a timeout block")
	p.chunkBlocksProposeNotEnoughTotal.Inc()
	return nil, nil
}

func (p *ChunkProposer) calculateChunkMetrics(chunk *encoding.Chunk) (*chunkMetrics, error) {
	var err error
	metrics := &chunkMetrics{
		txNum:               chunk.NumTransactions(),
		numBlocks:           uint64(len(chunk.Blocks)),
		firstBlockTimestamp: chunk.Blocks[0].Header.Time,
	}
	metrics.crcMax, err = chunk.CrcMax()
	if err != nil {
		return metrics, fmt.Errorf("failed to get crc max: %w", err)
	}
	firstBlockNum := chunk.Blocks[0].Header.Number.Uint64()
	if firstBlockNum >= p.banachForkHeight { // codecv1
		metrics.l1CommitBlobSize, err = codecv1.EstimateChunkL1CommitBlobSize(chunk)
		if err != nil {
			return metrics, fmt.Errorf("failed to estimate chunk L1 commit blob size: %w", err)
		}
	} else { // codecv0
		metrics.l1CommitCalldataSize, err = codecv0.EstimateChunkL1CommitCalldataSize(chunk)
		if err != nil {
			return metrics, fmt.Errorf("failed to estimate chunk L1 commit calldata size: %w", err)
		}
		metrics.l1CommitGas, err = codecv0.EstimateChunkL1CommitGas(chunk)
		if err != nil {
			return metrics, fmt.Errorf("failed to estimate chunk L1 commit gas: %w", err)
		}
	}
	return metrics, nil
}

func (p *ChunkProposer) recordChunkMetrics(metrics *chunkMetrics) {
	p.chunkTxNum.Set(float64(metrics.txNum))
	p.maxTxConsumption.Set(float64(metrics.crcMax))
	p.chunkBlocksNum.Set(float64(metrics.numBlocks))
	p.totalL1CommitCalldataSize.Set(float64(metrics.l1CommitCalldataSize))
	p.chunkEstimateL1CommitGas.Set(float64(metrics.l1CommitGas))
	p.totalL1CommitBlobSize.Set(float64(metrics.l1CommitBlobSize))
}
