package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types/encoding"
	"scroll-tech/common/types/encoding/codecv0"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
)

// BatchProposer proposes batches based on available unbatched chunks.
type BatchProposer struct {
	ctx context.Context
	db  *gorm.DB

	batchOrm   *orm.Batch
	chunkOrm   *orm.Chunk
	l2BlockOrm *orm.L2Block

	maxChunkNumPerBatch             uint64
	maxL1CommitGasPerBatch          uint64
	maxL1CommitCalldataSizePerBatch uint64
	batchTimeoutSec                 uint64
	gasCostIncreaseMultiplier       float64

	batchProposerCircleTotal           prometheus.Counter
	proposeBatchFailureTotal           prometheus.Counter
	proposeBatchUpdateInfoTotal        prometheus.Counter
	proposeBatchUpdateInfoFailureTotal prometheus.Counter
	totalL1CommitGas                   prometheus.Gauge
	totalL1CommitCalldataSize          prometheus.Gauge
	batchChunksNum                     prometheus.Gauge
	batchFirstBlockTimeoutReached      prometheus.Counter
	batchChunksProposeNotEnoughTotal   prometheus.Counter
}

// NewBatchProposer creates a new BatchProposer instance.
func NewBatchProposer(ctx context.Context, cfg *config.BatchProposerConfig, db *gorm.DB, reg prometheus.Registerer) *BatchProposer {
	log.Debug("new batch proposer",
		"maxChunkNumPerBatch", cfg.MaxChunkNumPerBatch,
		"maxL1CommitGasPerBatch", cfg.MaxL1CommitGasPerBatch,
		"maxL1CommitCalldataSizePerBatch", cfg.MaxL1CommitCalldataSizePerBatch,
		"batchTimeoutSec", cfg.BatchTimeoutSec,
		"gasCostIncreaseMultiplier", cfg.GasCostIncreaseMultiplier)

	return &BatchProposer{
		ctx:                             ctx,
		db:                              db,
		batchOrm:                        orm.NewBatch(db),
		chunkOrm:                        orm.NewChunk(db),
		l2BlockOrm:                      orm.NewL2Block(db),
		maxChunkNumPerBatch:             cfg.MaxChunkNumPerBatch,
		maxL1CommitGasPerBatch:          cfg.MaxL1CommitGasPerBatch,
		maxL1CommitCalldataSizePerBatch: cfg.MaxL1CommitCalldataSizePerBatch,
		batchTimeoutSec:                 cfg.BatchTimeoutSec,
		gasCostIncreaseMultiplier:       cfg.GasCostIncreaseMultiplier,

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
		totalL1CommitGas: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_batch_total_l1_commit_gas",
			Help: "The total l1 commit gas",
		}),
		totalL1CommitCalldataSize: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_batch_total_l1_call_data_size",
			Help: "The total l1 call data size",
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
	}
}

// TryProposeBatch tries to propose a new batches.
func (p *BatchProposer) TryProposeBatch() {
	p.batchProposerCircleTotal.Inc()
	batch, err := p.proposeBatch()
	if err != nil {
		p.proposeBatchFailureTotal.Inc()
		log.Error("proposeBatchChunks failed", "err", err)
		return
	}
	if batch == nil {
		return
	}
	err = p.db.Transaction(func(dbTX *gorm.DB) error {
		batch, dbErr := p.batchOrm.InsertBatch(p.ctx, batch, dbTX)
		if dbErr != nil {
			log.Warn("BatchProposer.updateBatchInfoInDB insert batch failure",
				"start chunk index", batch.StartChunkIndex, "end chunk index", batch.EndChunkIndex, "error", dbErr)
			return dbErr
		}
		dbErr = p.chunkOrm.UpdateBatchHashInRange(p.ctx, batch.StartChunkIndex, batch.EndChunkIndex, batch.Hash, dbTX)
		if dbErr != nil {
			log.Warn("BatchProposer.UpdateBatchHashInRange update the chunk's batch hash failure", "hash", batch.Hash, "error", dbErr)
			return dbErr
		}
		return nil
	})
	if err != nil {
		p.proposeBatchUpdateInfoFailureTotal.Inc()
		log.Error("update batch info in db failed", "err", err)
	}
}

func (p *BatchProposer) proposeBatch() (*encoding.Batch, error) {
	unbatchedChunkIndex, err := p.batchOrm.GetFirstUnbatchedChunkIndex(p.ctx)
	if err != nil {
		return nil, err
	}

	// select at most p.maxChunkNumPerBatch chunks
	dbChunks, err := p.chunkOrm.GetChunksGEIndex(p.ctx, unbatchedChunkIndex, int(p.maxChunkNumPerBatch))
	if err != nil {
		return nil, err
	}

	if len(dbChunks) == 0 {
		return nil, nil
	}

	daChunks, err := p.getDAChunks(dbChunks)
	if err != nil {
		return nil, err
	}

	parentDBBatch, err := p.batchOrm.GetLatestBatch(p.ctx)
	if err != nil {
		return nil, err
	}

	var batch encoding.Batch
	if parentDBBatch != nil {
		batch.Index = parentDBBatch.Index + 1
		parentDABatch, err := codecv0.NewDABatchFromBytes(parentDBBatch.BatchHeader)
		if err != nil {
			return nil, err
		}
		batch.TotalL1MessagePoppedBefore = parentDABatch.TotalL1MessagePopped
		batch.ParentBatchHash = common.HexToHash(parentDBBatch.Hash)
	}

	for i, chunk := range daChunks {
		batch.Chunks = append(batch.Chunks, chunk)
		totalL1CommitCalldataSize := codecv0.EstimateBatchL1CommitCalldataSize(&batch)
		totalL1CommitGas := codecv0.EstimateBatchL1CommitGas(&batch)
		totalOverEstimateL1CommitGas := uint64(p.gasCostIncreaseMultiplier * float64(totalL1CommitGas))
		if totalL1CommitCalldataSize > p.maxL1CommitCalldataSizePerBatch ||
			totalOverEstimateL1CommitGas > p.maxL1CommitGasPerBatch {
			// Check if the first chunk breaks hard limits.
			// If so, it indicates there are bugs in chunk-proposer, manual fix is needed.
			if i == 0 {
				if totalOverEstimateL1CommitGas > p.maxL1CommitGasPerBatch {
					return nil, fmt.Errorf(
						"the first chunk exceeds l1 commit gas limit; start block number: %v, end block number: %v, commit gas: %v, max commit gas limit: %v",
						dbChunks[0].StartBlockNumber,
						dbChunks[0].EndBlockNumber,
						totalL1CommitGas,
						p.maxL1CommitGasPerBatch,
					)
				}
				if totalL1CommitCalldataSize > p.maxL1CommitCalldataSizePerBatch {
					return nil, fmt.Errorf(
						"the first chunk exceeds l1 commit calldata size limit; start block number: %v, end block number %v, calldata size: %v, max calldata size limit: %v",
						dbChunks[0].StartBlockNumber,
						dbChunks[0].EndBlockNumber,
						totalL1CommitCalldataSize,
						p.maxL1CommitCalldataSizePerBatch,
					)
				}
			}

			log.Debug("breaking limit condition in batching",
				"currentL1CommitCalldataSize", totalL1CommitCalldataSize,
				"maxL1CommitCalldataSizePerBatch", p.maxL1CommitCalldataSizePerBatch,
				"currentOverEstimateL1CommitGas", totalOverEstimateL1CommitGas,
				"maxL1CommitGasPerBatch", p.maxL1CommitGasPerBatch)

			batch.Chunks = batch.Chunks[:len(batch.Chunks)-1]
			batch.StartChunkIndex = dbChunks[0].Index
			batch.EndChunkIndex = dbChunks[batch.GetChunkNum()-1].Index
			batch.StartChunkHash = common.HexToHash(dbChunks[0].Hash)
			batch.EndChunkHash = common.HexToHash(dbChunks[batch.GetChunkNum()-1].Hash)

			p.totalL1CommitGas.Set(float64(codecv0.EstimateBatchL1CommitGas(&batch)))
			p.totalL1CommitCalldataSize.Set(float64(codecv0.EstimateBatchL1CommitCalldataSize(&batch)))
			p.batchChunksNum.Set(float64(batch.GetChunkNum()))
			return &batch, nil
		}
	}

	currentTimeSec := uint64(time.Now().Unix())
	if dbChunks[0].StartBlockTime+p.batchTimeoutSec < currentTimeSec ||
		batch.GetChunkNum() == p.maxChunkNumPerBatch {
		if dbChunks[0].StartBlockTime+p.batchTimeoutSec < currentTimeSec {
			log.Warn("first block timeout",
				"start block number", dbChunks[0].StartBlockNumber,
				"start block timestamp", dbChunks[0].StartBlockTime,
				"current time", currentTimeSec,
			)
		} else {
			log.Info("reached maximum number of chunks in batch",
				"chunk count", batch.GetChunkNum(),
			)
		}

		p.batchFirstBlockTimeoutReached.Inc()
		p.totalL1CommitGas.Set(float64(codecv0.EstimateBatchL1CommitGas(&batch)))
		p.totalL1CommitCalldataSize.Set(float64(codecv0.EstimateBatchL1CommitCalldataSize(&batch)))
		p.batchChunksNum.Set(float64(batch.GetChunkNum()))

		batch.StartChunkIndex = dbChunks[0].Index
		batch.EndChunkIndex = dbChunks[batch.GetChunkNum()-1].Index
		batch.StartChunkHash = common.HexToHash(dbChunks[0].Hash)
		batch.EndChunkHash = common.HexToHash(dbChunks[batch.GetChunkNum()-1].Hash)

		return &batch, nil
	}

	log.Debug("pending chunks do not reach one of the constraints or contain a timeout block")
	p.batchChunksProposeNotEnoughTotal.Inc()
	return nil, nil
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
