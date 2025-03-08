package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
	"scroll-tech/rollup/internal/utils"
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
	maxUncompressedBatchBytesSize   uint64

	minCodecVersion encoding.CodecVersion
	chainCfg        *params.ChainConfig

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
	chunkEstimateGasTime               prometheus.Gauge
	chunkEstimateCalldataSizeTime      prometheus.Gauge
	chunkEstimateBlobSizeTime          prometheus.Gauge

	// total number of times that chunk proposer stops early due to compressed data compatibility breach
	compressedDataCompatibilityBreachTotal prometheus.Counter

	chunkProposeBlockHeight prometheus.Gauge
	chunkProposeThroughput  prometheus.Counter
}

// NewChunkProposer creates a new ChunkProposer instance.
func NewChunkProposer(ctx context.Context, cfg *config.ChunkProposerConfig, minCodecVersion encoding.CodecVersion, chainCfg *params.ChainConfig, db *gorm.DB, reg prometheus.Registerer) *ChunkProposer {
	log.Info("new chunk proposer",
		"maxBlockNumPerChunk", cfg.MaxBlockNumPerChunk,
		"maxTxNumPerChunk", cfg.MaxTxNumPerChunk,
		"maxL1CommitGasPerChunk", cfg.MaxL1CommitGasPerChunk,
		"maxL1CommitCalldataSizePerChunk", cfg.MaxL1CommitCalldataSizePerChunk,
		"maxRowConsumptionPerChunk", cfg.MaxRowConsumptionPerChunk,
		"chunkTimeoutSec", cfg.ChunkTimeoutSec,
		"gasCostIncreaseMultiplier", cfg.GasCostIncreaseMultiplier,
		"maxBlobSize", maxBlobSize,
		"maxUncompressedBatchBytesSize", cfg.MaxUncompressedBatchBytesSize)

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
		maxUncompressedBatchBytesSize:   cfg.MaxUncompressedBatchBytesSize,
		minCodecVersion:                 minCodecVersion,
		chainCfg:                        chainCfg,

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
		compressedDataCompatibilityBreachTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_chunk_due_to_compressed_data_compatibility_breach_total",
			Help: "Total number of propose chunk due to compressed data compatibility breach.",
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
		chunkEstimateGasTime: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_chunk_estimate_gas_time",
			Help: "Time taken to estimate gas for the chunk.",
		}),
		chunkEstimateCalldataSizeTime: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_chunk_estimate_calldata_size_time",
			Help: "Time taken to estimate calldata size for the chunk.",
		}),
		chunkEstimateBlobSizeTime: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_chunk_estimate_blob_size_time",
			Help: "Time taken to estimate blob size for the chunk.",
		}),
		chunkProposeBlockHeight: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_chunk_propose_block_height",
			Help: "The block height of the latest proposed chunk",
		}),
		chunkProposeThroughput: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_chunk_propose_throughput",
			Help: "The total gas used in proposed chunks",
		}),
	}

	return p
}

// TryProposeChunk tries to propose a new chunk.
func (p *ChunkProposer) TryProposeChunk() {
	p.chunkProposerCircleTotal.Inc()
	if err := p.proposeChunk(); err != nil {
		p.proposeChunkFailureTotal.Inc()
		log.Error("propose new chunk failed", "err", err)
		return
	}
}

func (p *ChunkProposer) updateDBChunkInfo(chunk *encoding.Chunk, codecVersion encoding.CodecVersion, metrics *utils.ChunkMetrics) error {
	if chunk == nil {
		return nil
	}

	compatibilityBreachOccurred := false

	for {
		compatible, err := encoding.CheckChunkCompressedDataCompatibility(chunk, codecVersion)
		if err != nil {
			log.Error("Failed to check chunk compressed data compatibility", "start block number", chunk.Blocks[0].Header.Number, "codecVersion", codecVersion, "err", err)
			return err
		}

		if compatible {
			break
		}

		compatibilityBreachOccurred = true

		if len(chunk.Blocks) == 1 {
			log.Warn("Disable compression: cannot truncate chunk with only 1 block for compatibility", "block number", chunk.Blocks[0].Header.Number)
			break
		}

		chunk.Blocks = chunk.Blocks[:len(chunk.Blocks)-1]

		log.Info("Chunk not compatible with compressed data, removing last block", "start block number", chunk.Blocks[0].Header.Number, "truncated block length", len(chunk.Blocks))
	}

	if compatibilityBreachOccurred {
		p.compressedDataCompatibilityBreachTotal.Inc()

		// recalculate chunk metrics after truncation
		var calcErr error
		metrics, calcErr = utils.CalculateChunkMetrics(chunk, codecVersion)
		if calcErr != nil {
			return fmt.Errorf("failed to calculate chunk metrics, start block number: %v, error: %w", chunk.Blocks[0].Header.Number, calcErr)
		}

		p.recordTimerChunkMetrics(metrics)
		p.recordAllChunkMetrics(metrics)
	}

	if len(chunk.Blocks) > 0 {
		p.chunkProposeBlockHeight.Set(float64(chunk.Blocks[len(chunk.Blocks)-1].Header.Number.Uint64()))
	}
	p.chunkProposeThroughput.Add(float64(chunk.TotalGasUsed()))

	p.proposeChunkUpdateInfoTotal.Inc()
	err := p.db.Transaction(func(dbTX *gorm.DB) error {
		dbChunk, err := p.chunkOrm.InsertChunk(p.ctx, chunk, codecVersion, *metrics, dbTX)
		if err != nil {
			log.Warn("ChunkProposer.InsertChunk failed", "codec version", codecVersion, "err", err)
			return err
		}
		if err := p.l2BlockOrm.UpdateChunkHashInRange(p.ctx, dbChunk.StartBlockNumber, dbChunk.EndBlockNumber, dbChunk.Hash, dbTX); err != nil {
			log.Error("failed to update chunk_hash for l2_blocks", "chunk hash", dbChunk.Hash, "start block", dbChunk.StartBlockNumber, "end block", dbChunk.EndBlockNumber, "err", err)
			return err
		}
		return nil
	})
	if err != nil {
		p.proposeChunkUpdateInfoFailureTotal.Inc()
		log.Error("update chunk info in orm failed", "err", err)
		return err
	}
	return nil
}

func (p *ChunkProposer) proposeChunk() error {
	// unchunkedBlockHeight >= 1, assuming genesis batch with chunk 0, block 0 is committed.
	unchunkedBlockHeight, err := p.chunkOrm.GetUnchunkedBlockHeight(p.ctx)
	if err != nil {
		return err
	}

	maxBlocksThisChunk := p.maxBlockNumPerChunk

	// select at most maxBlocksThisChunk blocks
	blocks, err := p.l2BlockOrm.GetL2BlocksGEHeight(p.ctx, unchunkedBlockHeight, int(maxBlocksThisChunk))
	if err != nil {
		return err
	}

	if len(blocks) == 0 {
		return nil
	}

	// Ensure all blocks in the same chunk use the same hardfork name
	// If a different hardfork name is found, truncate the blocks slice at that point
	hardforkName := encoding.GetHardforkName(p.chainCfg, blocks[0].Header.Number.Uint64(), blocks[0].Header.Time)
	for i := 1; i < len(blocks); i++ {
		currentHardfork := encoding.GetHardforkName(p.chainCfg, blocks[i].Header.Number.Uint64(), blocks[i].Header.Time)
		if currentHardfork != hardforkName {
			blocks = blocks[:i]
			maxBlocksThisChunk = uint64(i) // update maxBlocksThisChunk to trigger chunking, because these blocks are the last blocks before the hardfork
			break
		}
	}

	codecVersion := encoding.GetCodecVersion(p.chainCfg, blocks[0].Header.Number.Uint64(), blocks[0].Header.Time)

	if codecVersion < p.minCodecVersion {
		return fmt.Errorf("unsupported codec version: %v, expected at least %v", codecVersion, p.minCodecVersion)
	}

	// Including Curie block in a sole chunk.
	if p.chainCfg.CurieBlock != nil && blocks[0].Header.Number.Cmp(p.chainCfg.CurieBlock) == 0 {
		chunk := encoding.Chunk{Blocks: blocks[:1]}
		metrics, calcErr := utils.CalculateChunkMetrics(&chunk, codecVersion)
		if calcErr != nil {
			return fmt.Errorf("failed to calculate chunk metrics: %w", calcErr)
		}
		p.recordTimerChunkMetrics(metrics)
		return p.updateDBChunkInfo(&chunk, codecVersion, metrics)
	}

	var chunk encoding.Chunk
	chunk.Blocks = make([]*encoding.Block, 0, len(blocks))
	for i, block := range blocks {
		chunk.Blocks = append(chunk.Blocks, block)

		metrics, calcErr := utils.CalculateChunkMetrics(&chunk, codecVersion)
		if calcErr != nil {
			return fmt.Errorf("failed to calculate chunk metrics: %w", calcErr)
		}

		p.recordTimerChunkMetrics(metrics)

		overEstimatedL1CommitGas := uint64(p.gasCostIncreaseMultiplier * float64(metrics.L1CommitGas))
		if metrics.TxNum > p.maxTxNumPerChunk ||
			metrics.L1CommitCalldataSize > p.maxL1CommitCalldataSizePerChunk ||
			overEstimatedL1CommitGas > p.maxL1CommitGasPerChunk ||
			metrics.CrcMax > p.maxRowConsumptionPerChunk ||
			metrics.L1CommitBlobSize > maxBlobSize ||
			metrics.L1CommitUncompressedBatchBytesSize > p.maxUncompressedBatchBytesSize {
			if i == 0 {
				// The first block exceeds hard limits, which indicates a bug in the sequencer, manual fix is needed.
				return fmt.Errorf("the first block exceeds limits; block number: %v, limits: %+v, maxTxNum: %v, maxL1CommitCalldataSize: %v, maxL1CommitGas: %v, maxRowConsumption: %v, maxBlobSize: %v, maxUncompressedBatchBytesSize: %v",
					block.Header.Number, metrics, p.maxTxNumPerChunk, p.maxL1CommitCalldataSizePerChunk, p.maxL1CommitGasPerChunk, p.maxRowConsumptionPerChunk, maxBlobSize, p.maxUncompressedBatchBytesSize)
			}

			log.Debug("breaking limit condition in chunking",
				"txNum", metrics.TxNum,
				"maxTxNum", p.maxTxNumPerChunk,
				"l1CommitCalldataSize", metrics.L1CommitCalldataSize,
				"maxL1CommitCalldataSize", p.maxL1CommitCalldataSizePerChunk,
				"l1CommitGas", metrics.L1CommitGas,
				"overEstimatedL1CommitGas", overEstimatedL1CommitGas,
				"maxL1CommitGas", p.maxL1CommitGasPerChunk,
				"rowConsumption", metrics.CrcMax,
				"maxRowConsumption", p.maxRowConsumptionPerChunk,
				"l1CommitBlobSize", metrics.L1CommitBlobSize,
				"maxBlobSize", maxBlobSize,
				"L1CommitUncompressedBatchBytesSize", metrics.L1CommitUncompressedBatchBytesSize,
				"maxUncompressedBatchBytesSize", p.maxUncompressedBatchBytesSize)

			chunk.Blocks = chunk.Blocks[:len(chunk.Blocks)-1]

			metrics, calcErr := utils.CalculateChunkMetrics(&chunk, codecVersion)
			if calcErr != nil {
				return fmt.Errorf("failed to calculate chunk metrics: %w", calcErr)
			}

			p.recordAllChunkMetrics(metrics)
			return p.updateDBChunkInfo(&chunk, codecVersion, metrics)
		}
	}

	metrics, calcErr := utils.CalculateChunkMetrics(&chunk, codecVersion)
	if calcErr != nil {
		return fmt.Errorf("failed to calculate chunk metrics: %w", calcErr)
	}

	currentTimeSec := uint64(time.Now().Unix())
	if metrics.FirstBlockTimestamp+p.chunkTimeoutSec < currentTimeSec || metrics.NumBlocks == maxBlocksThisChunk {
		log.Info("reached maximum number of blocks in chunk or first block timeout",
			"block count", len(chunk.Blocks),
			"start block number", chunk.Blocks[0].Header.Number,
			"start block timestamp", metrics.FirstBlockTimestamp,
			"current time", currentTimeSec)

		p.chunkFirstBlockTimeoutReached.Inc()
		p.recordAllChunkMetrics(metrics)
		return p.updateDBChunkInfo(&chunk, codecVersion, metrics)
	}

	log.Debug("pending blocks do not reach one of the constraints or contain a timeout block")
	p.recordTimerChunkMetrics(metrics)
	p.chunkBlocksProposeNotEnoughTotal.Inc()
	return nil
}

func (p *ChunkProposer) recordAllChunkMetrics(metrics *utils.ChunkMetrics) {
	p.chunkTxNum.Set(float64(metrics.TxNum))
	p.maxTxConsumption.Set(float64(metrics.CrcMax))
	p.chunkBlocksNum.Set(float64(metrics.NumBlocks))
	p.totalL1CommitCalldataSize.Set(float64(metrics.L1CommitCalldataSize))
	p.chunkEstimateL1CommitGas.Set(float64(metrics.L1CommitGas))
	p.totalL1CommitBlobSize.Set(float64(metrics.L1CommitBlobSize))
	p.chunkEstimateGasTime.Set(float64(metrics.EstimateGasTime))
	p.chunkEstimateCalldataSizeTime.Set(float64(metrics.EstimateCalldataSizeTime))
	p.chunkEstimateBlobSizeTime.Set(float64(metrics.EstimateBlobSizeTime))
}

func (p *ChunkProposer) recordTimerChunkMetrics(metrics *utils.ChunkMetrics) {
	p.chunkEstimateGasTime.Set(float64(metrics.EstimateGasTime))
	p.chunkEstimateCalldataSizeTime.Set(float64(metrics.EstimateCalldataSizeTime))
	p.chunkEstimateBlobSizeTime.Set(float64(metrics.EstimateBlobSizeTime))
}
