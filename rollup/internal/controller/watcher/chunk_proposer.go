package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
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

	cfg *config.ChunkProposerConfig

	replayMode      bool
	minCodecVersion encoding.CodecVersion
	chainCfg        *params.ChainConfig

	chunkProposerCircleTotal           prometheus.Counter
	proposeChunkFailureTotal           prometheus.Counter
	proposeChunkUpdateInfoTotal        prometheus.Counter
	proposeChunkUpdateInfoFailureTotal prometheus.Counter
	chunkTxNum                         prometheus.Gauge
	chunkL2Gas                         prometheus.Gauge
	totalL1CommitBlobSize              prometheus.Gauge
	chunkBlocksNum                     prometheus.Gauge
	chunkFirstBlockTimeoutReached      prometheus.Counter
	chunkBlocksProposeNotEnoughTotal   prometheus.Counter
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
		"maxL2GasPerChunk", cfg.MaxL2GasPerChunk,
		"chunkTimeoutSec", cfg.ChunkTimeoutSec,
		"maxBlobSize", maxBlobSize)

	p := &ChunkProposer{
		ctx:             ctx,
		db:              db,
		chunkOrm:        orm.NewChunk(db),
		l2BlockOrm:      orm.NewL2Block(db),
		cfg:             cfg,
		replayMode:      false,
		minCodecVersion: minCodecVersion,
		chainCfg:        chainCfg,

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
		chunkL2Gas: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_chunk_l2_gas",
			Help: "The chunk l2 gas",
		}),
		totalL1CommitBlobSize: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_chunk_total_l1_commit_blob_size",
			Help: "The total l1 commit blob size",
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

// SetReplayDB sets the replay database for the ChunkProposer.
// This is used for the proposer tool only, to change the l2_block data source.
// This function is not thread-safe and should be called after initializing the ChunkProposer and before starting to propose chunks.
func (p *ChunkProposer) SetReplayDB(replayDB *gorm.DB) {
	p.l2BlockOrm = orm.NewL2Block(replayDB)
	p.replayMode = true
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
	if chunk == nil || len(chunk.Blocks) == 0 {
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
		chunk.PostL1MessageQueueHash, err = encoding.MessageQueueV2ApplyL1MessagesFromBlocks(chunk.PrevL1MessageQueueHash, chunk.Blocks)
		if err != nil {
			log.Error("Failed to calculate last L1 message queue hash for block", "block number", chunk.Blocks[0].Header.Number, "err", err)
			return err
		}

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

	p.chunkProposeBlockHeight.Set(float64(chunk.Blocks[len(chunk.Blocks)-1].Header.Number.Uint64()))
	p.chunkProposeThroughput.Add(float64(chunk.TotalGasUsed()))

	p.proposeChunkUpdateInfoTotal.Inc()
	err := p.db.Transaction(func(dbTX *gorm.DB) error {
		dbChunk, err := p.chunkOrm.InsertChunk(p.ctx, chunk, codecVersion, *metrics, dbTX)
		if err != nil {
			log.Warn("ChunkProposer.InsertChunk failed", "codec version", codecVersion, "err", err)
			return err
		}
		// In replayMode we don't need to update chunk_hash in l2_block table.
		if !p.replayMode {
			if err := p.l2BlockOrm.UpdateChunkHashInRange(p.ctx, dbChunk.StartBlockNumber, dbChunk.EndBlockNumber, dbChunk.Hash, dbTX); err != nil {
				log.Error("failed to update chunk_hash for l2_block", "chunk hash", dbChunk.Hash, "start block", dbChunk.StartBlockNumber, "end block", dbChunk.EndBlockNumber, "err", err)
				return err
			}
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

	maxBlocksThisChunk := p.cfg.MaxBlockNumPerChunk

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

	var chunk encoding.Chunk
	parentChunk, err := p.chunkOrm.GetLatestChunk(p.ctx)
	if err != nil || parentChunk == nil {
		return fmt.Errorf("failed to get parent chunk: %w", err)
	}

	chunk.PrevL1MessageQueueHash = common.HexToHash(parentChunk.PostL1MessageQueueHash)

	// previous chunk is not CodecV7, this means this is the first chunk of the fork.
	if encoding.CodecVersion(parentChunk.CodecVersion) < codecVersion {
		chunk.PrevL1MessageQueueHash = common.Hash{}
	}

	chunk.PostL1MessageQueueHash = chunk.PrevL1MessageQueueHash

	var previousPostL1MessageQueueHash common.Hash
	chunk.Blocks = make([]*encoding.Block, 0, len(blocks))
	for i, block := range blocks {
		chunk.Blocks = append(chunk.Blocks, block)

		previousPostL1MessageQueueHash = chunk.PostL1MessageQueueHash
		chunk.PostL1MessageQueueHash, err = encoding.MessageQueueV2ApplyL1MessagesFromBlocks(previousPostL1MessageQueueHash, []*encoding.Block{block})
		if err != nil {
			return fmt.Errorf("failed to calculate last L1 message queue hash for block %d: %w", block.Header.Number.Uint64(), err)
		}

		metrics, calcErr := utils.CalculateChunkMetrics(&chunk, codecVersion)
		if calcErr != nil {
			return fmt.Errorf("failed to calculate chunk metrics: %w", calcErr)
		}

		p.recordTimerChunkMetrics(metrics)

		if metrics.L2Gas > p.cfg.MaxL2GasPerChunk || metrics.L1CommitBlobSize > maxBlobSize || metrics.L1CommitUncompressedBatchBytesSize > p.cfg.MaxUncompressedBatchBytesSize {
			if i == 0 {
				// The first block exceeds hard limits, which indicates a bug in the sequencer, manual fix is needed.
				return fmt.Errorf("the first block exceeds limits; block number: %v, limits: %+v, maxBlobSize: %v, maxUncompressedBatchBytesSize: %v", block.Header.Number, metrics, maxBlobSize, p.cfg.MaxUncompressedBatchBytesSize)
			}

			log.Debug("breaking limit condition in chunking",
				"l2Gas", metrics.L2Gas,
				"maxL2Gas", p.cfg.MaxL2GasPerChunk,
				"l1CommitBlobSize", metrics.L1CommitBlobSize,
				"maxBlobSize", maxBlobSize,
				"L1CommitUncompressedBatchBytesSize", metrics.L1CommitUncompressedBatchBytesSize,
				"maxUncompressedBatchBytesSize", p.cfg.MaxUncompressedBatchBytesSize)

			chunk.Blocks = chunk.Blocks[:len(chunk.Blocks)-1]
			chunk.PostL1MessageQueueHash = previousPostL1MessageQueueHash

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
	if metrics.FirstBlockTimestamp+p.cfg.ChunkTimeoutSec < currentTimeSec || metrics.NumBlocks == maxBlocksThisChunk {
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
	p.chunkBlocksNum.Set(float64(metrics.NumBlocks))
	p.chunkL2Gas.Set(float64(metrics.L2Gas))
	p.totalL1CommitBlobSize.Set(float64(metrics.L1CommitBlobSize))
	p.chunkEstimateBlobSizeTime.Set(float64(metrics.EstimateBlobSizeTime))
}

func (p *ChunkProposer) recordTimerChunkMetrics(metrics *utils.ChunkMetrics) {
	p.chunkEstimateBlobSizeTime.Set(float64(metrics.EstimateBlobSizeTime))
}
