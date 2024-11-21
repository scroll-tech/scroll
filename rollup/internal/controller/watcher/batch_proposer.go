package watcher

import (
	"context"
	"fmt"
	"math/big"
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

// BatchProposer proposes batches based on available unbatched chunks.
type BatchProposer struct {
	ctx context.Context
	db  *gorm.DB

	batchOrm   *orm.Batch
	chunkOrm   *orm.Chunk
	l2BlockOrm *orm.L2Block

	maxL1CommitGasPerBatch          uint64
	maxL1CommitCalldataSizePerBatch uint64
	batchTimeoutSec                 uint64
	gasCostIncreaseMultiplier       float64
	maxUncompressedBatchBytesSize   uint64

	minCodecVersion encoding.CodecVersion
	chainCfg        *params.ChainConfig

	batchProposerCircleTotal           prometheus.Counter
	proposeBatchFailureTotal           prometheus.Counter
	proposeBatchUpdateInfoTotal        prometheus.Counter
	proposeBatchUpdateInfoFailureTotal prometheus.Counter
	totalL1CommitGas                   prometheus.Gauge
	totalL1CommitCalldataSize          prometheus.Gauge
	totalL1CommitBlobSize              prometheus.Gauge
	batchChunksNum                     prometheus.Gauge
	batchFirstBlockTimeoutReached      prometheus.Counter
	batchChunksProposeNotEnoughTotal   prometheus.Counter
	batchEstimateGasTime               prometheus.Gauge
	batchEstimateCalldataSizeTime      prometheus.Gauge
	batchEstimateBlobSizeTime          prometheus.Gauge

	// total number of times that batch proposer stops early due to compressed data compatibility breach
	compressedDataCompatibilityBreachTotal prometheus.Counter

	batchProposeBlockHeight prometheus.Gauge
	batchProposeThroughput  prometheus.Counter
}

// NewBatchProposer creates a new BatchProposer instance.
func NewBatchProposer(ctx context.Context, cfg *config.BatchProposerConfig, minCodecVersion encoding.CodecVersion, chainCfg *params.ChainConfig, db *gorm.DB, reg prometheus.Registerer) *BatchProposer {
	log.Info("new batch proposer",
		"maxL1CommitGasPerBatch", cfg.MaxL1CommitGasPerBatch,
		"maxL1CommitCalldataSizePerBatch", cfg.MaxL1CommitCalldataSizePerBatch,
		"batchTimeoutSec", cfg.BatchTimeoutSec,
		"gasCostIncreaseMultiplier", cfg.GasCostIncreaseMultiplier,
		"maxBlobSize", maxBlobSize,
		"maxUncompressedBatchBytesSize", cfg.MaxUncompressedBatchBytesSize)

	p := &BatchProposer{
		ctx:                             ctx,
		db:                              db,
		batchOrm:                        orm.NewBatch(db),
		chunkOrm:                        orm.NewChunk(db),
		l2BlockOrm:                      orm.NewL2Block(db),
		maxL1CommitGasPerBatch:          cfg.MaxL1CommitGasPerBatch,
		maxL1CommitCalldataSizePerBatch: cfg.MaxL1CommitCalldataSizePerBatch,
		batchTimeoutSec:                 cfg.BatchTimeoutSec,
		gasCostIncreaseMultiplier:       cfg.GasCostIncreaseMultiplier,
		maxUncompressedBatchBytesSize:   cfg.MaxUncompressedBatchBytesSize,
		minCodecVersion:                 minCodecVersion,
		chainCfg:                        chainCfg,

		batchProposerCircleTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_batch_circle_total",
			Help: "Total number of propose batch total.",
		}),
		proposeBatchFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_batch_failure_circle_total",
			Help: "Total number of propose batch total.",
		}),
		proposeBatchUpdateInfoTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_batch_update_info_total",
			Help: "Total number of propose batch update info total.",
		}),
		proposeBatchUpdateInfoFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_batch_update_info_failure_total",
			Help: "Total number of propose batch update info failure total.",
		}),
		compressedDataCompatibilityBreachTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_batch_due_to_compressed_data_compatibility_breach_total",
			Help: "Total number of propose batch due to compressed data compatibility breach.",
		}),
		totalL1CommitGas: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_batch_total_l1_commit_gas",
			Help: "The total l1 commit gas",
		}),
		totalL1CommitCalldataSize: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_batch_total_l1_call_data_size",
			Help: "The total l1 call data size",
		}),
		totalL1CommitBlobSize: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_batch_total_l1_commit_blob_size",
			Help: "The total l1 commit blob size",
		}),
		batchChunksNum: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_batch_chunks_number",
			Help: "The number of chunks in the batch",
		}),
		batchFirstBlockTimeoutReached: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_batch_first_block_timeout_reached_total",
			Help: "Total times of batch's first block timeout reached",
		}),
		batchChunksProposeNotEnoughTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_batch_chunks_propose_not_enough_total",
			Help: "Total number of batch chunk propose not enough",
		}),
		batchEstimateGasTime: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_batch_estimate_gas_time",
			Help: "Time taken to estimate gas for the chunk.",
		}),
		batchEstimateCalldataSizeTime: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_batch_estimate_calldata_size_time",
			Help: "Time taken to estimate calldata size for the chunk.",
		}),
		batchEstimateBlobSizeTime: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_batch_estimate_blob_size_time",
			Help: "Time taken to estimate blob size for the chunk.",
		}),
		batchProposeBlockHeight: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_batch_propose_block_height",
			Help: "The block height of the latest proposed batch",
		}),
		batchProposeThroughput: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_batch_propose_throughput",
			Help: "The total gas used in proposed batches",
		}),
	}

	return p
}

// TryProposeBatch tries to propose a new batches.
func (p *BatchProposer) TryProposeBatch() {
	p.batchProposerCircleTotal.Inc()
	if err := p.proposeBatch(); err != nil {
		p.proposeBatchFailureTotal.Inc()
		log.Error("proposeBatchChunks failed", "err", err)
		return
	}
}

func (p *BatchProposer) updateDBBatchInfo(batch *encoding.Batch, codecVersion encoding.CodecVersion, metrics *utils.BatchMetrics) error {
	compatibilityBreachOccurred := false

	for {
		compatible, err := encoding.CheckBatchCompressedDataCompatibility(batch, codecVersion)
		if err != nil {
			log.Error("Failed to check batch compressed data compatibility", "batch index", batch.Index, "codecVersion", codecVersion, "err", err)
			return err
		}

		if compatible {
			break
		}

		compatibilityBreachOccurred = true

		if len(batch.Chunks) == 1 {
			log.Warn("Disable compression: cannot truncate batch with only 1 chunk for compatibility", "start block number", batch.Chunks[0].Blocks[0].Header.Number.Uint64(),
				"end block number", batch.Chunks[0].Blocks[len(batch.Chunks[0].Blocks)-1].Header.Number.Uint64())
			break
		}

		batch.Chunks = batch.Chunks[:len(batch.Chunks)-1]

		log.Info("Batch not compatible with compressed data, removing last chunk", "batch index", batch.Index, "truncated chunk length", len(batch.Chunks))
	}

	if compatibilityBreachOccurred {
		p.compressedDataCompatibilityBreachTotal.Inc()

		// recalculate batch metrics after truncation
		var calcErr error
		metrics, calcErr = utils.CalculateBatchMetrics(batch, codecVersion)
		if calcErr != nil {
			return fmt.Errorf("failed to calculate batch metrics, batch index: %v, error: %w", batch.Index, calcErr)
		}

		p.recordTimerBatchMetrics(metrics)
		p.recordAllBatchMetrics(metrics)
	}

	if len(batch.Chunks) > 0 && len(batch.Chunks[len(batch.Chunks)-1].Blocks) > 0 {
		lastChunk := batch.Chunks[len(batch.Chunks)-1]
		lastBlock := lastChunk.Blocks[len(lastChunk.Blocks)-1]
		p.batchProposeBlockHeight.Set(float64(lastBlock.Header.Number.Uint64()))
	}

	var totalGasUsed uint64
	for _, chunk := range batch.Chunks {
		totalGasUsed += chunk.TotalGasUsed()
	}
	p.batchProposeThroughput.Add(float64(totalGasUsed))

	p.proposeBatchUpdateInfoTotal.Inc()
	err := p.db.Transaction(func(dbTX *gorm.DB) error {
		dbBatch, dbErr := p.batchOrm.InsertBatch(p.ctx, batch, codecVersion, *metrics, dbTX)
		if dbErr != nil {
			log.Warn("BatchProposer.updateDBBatchInfo insert batch failure", "index", batch.Index, "parent hash", batch.ParentBatchHash.Hex(), "codec version", codecVersion, "error", dbErr)
			return dbErr
		}
		if dbErr = p.chunkOrm.UpdateBatchHashInRange(p.ctx, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex, dbBatch.Hash, dbTX); dbErr != nil {
			log.Warn("BatchProposer.UpdateBatchHashInRange update the chunk's batch hash failure", "hash", dbBatch.Hash, "error", dbErr)
			return dbErr
		}
		return nil
	})
	if err != nil {
		p.proposeBatchUpdateInfoFailureTotal.Inc()
		log.Error("update batch info in db failed", "err", err)
	}
	return nil
}

func (p *BatchProposer) proposeBatch() error {
	firstUnbatchedChunkIndex, err := p.batchOrm.GetFirstUnbatchedChunkIndex(p.ctx)
	if err != nil {
		return err
	}

	firstUnbatchedChunk, err := p.chunkOrm.GetChunkByIndex(p.ctx, firstUnbatchedChunkIndex)
	if err != nil || firstUnbatchedChunk == nil {
		return err
	}

	codec := encoding.CodecFromConfig(p.chainCfg, new(big.Int).SetUint64(firstUnbatchedChunk.StartBlockNumber), firstUnbatchedChunk.StartBlockTime)
	if codec == nil {
		return fmt.Errorf("failed to retrieve codec for block number %v and time %v", firstUnbatchedChunk.StartBlockNumber, firstUnbatchedChunk.StartBlockTime)
	}

	if codec.Version() < p.minCodecVersion {
		return fmt.Errorf("unsupported codec version: %v, expected at least %v", codec.Version(), p.minCodecVersion)
	}

	maxChunksThisBatch := codec.MaxNumChunksPerBatch()

	// select at most maxChunkNumPerBatch chunks
	dbChunks, err := p.chunkOrm.GetChunksGEIndex(p.ctx, firstUnbatchedChunkIndex, maxChunksThisBatch)
	if err != nil {
		return err
	}

	if len(dbChunks) == 0 {
		return nil
	}

	// Ensure all chunks in the same batch use the same hardfork name
	// If a different hardfork name is found, truncate the chunks slice at that point
	hardforkName := encoding.GetHardforkName(p.chainCfg, dbChunks[0].StartBlockNumber, dbChunks[0].StartBlockTime)
	for i := 1; i < len(dbChunks); i++ {
		currentHardfork := encoding.GetHardforkName(p.chainCfg, dbChunks[i].StartBlockNumber, dbChunks[i].StartBlockTime)
		if currentHardfork != hardforkName {
			dbChunks = dbChunks[:i]
			maxChunksThisBatch = len(dbChunks) // update maxChunksThisBatch to trigger batching, because these chunks are the last chunks before the hardfork
			break
		}
	}

	daChunks, err := p.getDAChunks(dbChunks)
	if err != nil {
		return err
	}

	dbParentBatch, err := p.batchOrm.GetLatestBatch(p.ctx)
	if err != nil {
		return err
	}

	var batch encoding.Batch
	batch.Index = dbParentBatch.Index + 1
	batch.ParentBatchHash = common.HexToHash(dbParentBatch.Hash)
	batch.TotalL1MessagePoppedBefore = firstUnbatchedChunk.TotalL1MessagesPoppedBefore

	for i, chunk := range daChunks {
		batch.Chunks = append(batch.Chunks, chunk)
		metrics, calcErr := utils.CalculateBatchMetrics(&batch, codec.Version())
		if calcErr != nil {
			return fmt.Errorf("failed to calculate batch metrics: %w", calcErr)
		}

		p.recordTimerBatchMetrics(metrics)

		totalOverEstimateL1CommitGas := uint64(p.gasCostIncreaseMultiplier * float64(metrics.L1CommitGas))
		if metrics.L1CommitCalldataSize > p.maxL1CommitCalldataSizePerBatch || totalOverEstimateL1CommitGas > p.maxL1CommitGasPerBatch ||
			metrics.L1CommitBlobSize > maxBlobSize || metrics.L1CommitUncompressedBatchBytesSize > p.maxUncompressedBatchBytesSize {
			if i == 0 {
				// The first chunk exceeds hard limits, which indicates a bug in the chunk-proposer, manual fix is needed.
				return fmt.Errorf("the first chunk exceeds limits; start block number: %v, end block number: %v, limits: %+v, maxChunkNum: %v, maxL1CommitCalldataSize: %v, maxL1CommitGas: %v, maxBlobSize: %v, maxUncompressedBatchBytesSize: %v",
					dbChunks[0].StartBlockNumber, dbChunks[0].EndBlockNumber, metrics, maxChunksThisBatch, p.maxL1CommitCalldataSizePerBatch, p.maxL1CommitGasPerBatch, maxBlobSize, p.maxUncompressedBatchBytesSize)
			}

			log.Debug("breaking limit condition in batching",
				"l1CommitCalldataSize", metrics.L1CommitCalldataSize,
				"maxL1CommitCalldataSize", p.maxL1CommitCalldataSizePerBatch,
				"l1CommitGas", metrics.L1CommitGas,
				"overEstimateL1CommitGas", totalOverEstimateL1CommitGas,
				"maxL1CommitGas", p.maxL1CommitGasPerBatch,
				"l1CommitBlobSize", metrics.L1CommitBlobSize,
				"maxBlobSize", maxBlobSize,
				"L1CommitUncompressedBatchBytesSize", metrics.L1CommitUncompressedBatchBytesSize,
				"maxUncompressedBatchBytesSize", p.maxUncompressedBatchBytesSize)

			batch.Chunks = batch.Chunks[:len(batch.Chunks)-1]

			metrics, err := utils.CalculateBatchMetrics(&batch, codec.Version())
			if err != nil {
				return fmt.Errorf("failed to calculate batch metrics: %w", err)
			}

			p.recordAllBatchMetrics(metrics)
			return p.updateDBBatchInfo(&batch, codec.Version(), metrics)
		}
	}

	metrics, calcErr := utils.CalculateBatchMetrics(&batch, codec.Version())
	if calcErr != nil {
		return fmt.Errorf("failed to calculate batch metrics: %w", calcErr)
	}
	currentTimeSec := uint64(time.Now().Unix())
	if metrics.FirstBlockTimestamp+p.batchTimeoutSec < currentTimeSec || metrics.NumChunks == uint64(maxChunksThisBatch) {
		log.Info("reached maximum number of chunks in batch or first block timeout",
			"chunk count", metrics.NumChunks,
			"start block number", dbChunks[0].StartBlockNumber,
			"start block timestamp", dbChunks[0].StartBlockTime,
			"current time", currentTimeSec)

		p.batchFirstBlockTimeoutReached.Inc()
		p.recordAllBatchMetrics(metrics)
		return p.updateDBBatchInfo(&batch, codec.Version(), metrics)
	}

	log.Debug("pending chunks do not reach one of the constraints or contain a timeout block")
	p.recordTimerBatchMetrics(metrics)
	p.batchChunksProposeNotEnoughTotal.Inc()
	return nil
}

func (p *BatchProposer) getDAChunks(dbChunks []*orm.Chunk) ([]*encoding.Chunk, error) {
	chunks := make([]*encoding.Chunk, len(dbChunks))
	for i, c := range dbChunks {
		blocks, err := p.l2BlockOrm.GetL2BlocksInRange(p.ctx, c.StartBlockNumber, c.EndBlockNumber)
		if err != nil {
			log.Error("Failed to fetch blocks", "start number", c.StartBlockNumber, "end number", c.EndBlockNumber, "error", err)
			return nil, err
		}
		chunks[i] = &encoding.Chunk{
			Blocks: blocks,
		}
	}
	return chunks, nil
}

func (p *BatchProposer) recordAllBatchMetrics(metrics *utils.BatchMetrics) {
	p.totalL1CommitGas.Set(float64(metrics.L1CommitGas))
	p.totalL1CommitCalldataSize.Set(float64(metrics.L1CommitCalldataSize))
	p.batchChunksNum.Set(float64(metrics.NumChunks))
	p.totalL1CommitBlobSize.Set(float64(metrics.L1CommitBlobSize))
	p.batchEstimateGasTime.Set(float64(metrics.EstimateGasTime))
	p.batchEstimateCalldataSizeTime.Set(float64(metrics.EstimateCalldataSizeTime))
	p.batchEstimateBlobSizeTime.Set(float64(metrics.EstimateBlobSizeTime))
}

func (p *BatchProposer) recordTimerBatchMetrics(metrics *utils.BatchMetrics) {
	p.batchEstimateGasTime.Set(float64(metrics.EstimateGasTime))
	p.batchEstimateCalldataSizeTime.Set(float64(metrics.EstimateCalldataSizeTime))
	p.batchEstimateBlobSizeTime.Set(float64(metrics.EstimateBlobSizeTime))
}
