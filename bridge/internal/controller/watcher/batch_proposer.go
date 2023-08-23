package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/orm"
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
	maxL1CommitCalldataSizePerBatch uint32
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
		"batchTimeoutSec", cfg.BatchTimeoutSec)

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
			Name: "bridge_propose_batch_circle_total",
			Help: "Total number of propose batch total.",
		}),
		proposeBatchFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_propose_batch_failure_circle_total",
			Help: "Total number of propose batch total.",
		}),
		proposeBatchUpdateInfoTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_propose_batch_update_info_total",
			Help: "Total number of propose batch update info total.",
		}),
		proposeBatchUpdateInfoFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_propose_batch_update_info_failure_total",
			Help: "Total number of propose batch update info failure total.",
		}),
		totalL1CommitGas: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "bridge_propose_batch_total_l1_commit_gas",
			Help: "The total l1 commit gas",
		}),
		totalL1CommitCalldataSize: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "bridge_propose_batch_total_l1_call_data_size",
			Help: "The total l1 call data size",
		}),
		batchChunksNum: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "bridge_propose_batch_chunks_number",
			Help: "The number of chunks in the batch",
		}),
		batchFirstBlockTimeoutReached: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_propose_batch_first_block_timeout_reached_total",
			Help: "Total times of batch's first block timeout reached",
		}),
		batchChunksProposeNotEnoughTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_propose_batch_chunks_propose_not_enough_total",
			Help: "Total number of batch chunk propose not enough",
		}),
	}
}

// TryProposeBatch tries to propose a new batches.
func (p *BatchProposer) TryProposeBatch() {
	p.batchProposerCircleTotal.Inc()
	dbChunks, err := p.proposeBatchChunks()
	if err != nil {
		p.proposeBatchFailureTotal.Inc()
		log.Error("proposeBatchChunks failed", "err", err)
		return
	}
	if err := p.updateBatchInfoInDB(dbChunks); err != nil {
		p.proposeBatchUpdateInfoFailureTotal.Inc()
		log.Error("update batch info in db failed", "err", err)
	}
}

func (p *BatchProposer) updateBatchInfoInDB(dbChunks []*orm.Chunk) error {
	p.proposeBatchUpdateInfoTotal.Inc()
	numChunks := len(dbChunks)
	if numChunks <= 0 {
		return nil
	}
	chunks, err := p.dbChunksToBridgeChunks(dbChunks)
	if err != nil {
		return err
	}

	startChunkIndex := dbChunks[0].Index
	startChunkHash := dbChunks[0].Hash
	endChunkIndex := dbChunks[numChunks-1].Index
	endChunkHash := dbChunks[numChunks-1].Hash
	err = p.db.Transaction(func(dbTX *gorm.DB) error {
		batch, dbErr := p.batchOrm.InsertBatch(p.ctx, startChunkIndex, endChunkIndex, startChunkHash, endChunkHash, chunks, dbTX)
		if dbErr != nil {
			log.Warn("BatchProposer.updateBatchInfoInDB insert batch failure", "error", "start chunk index", startChunkIndex, "end chunk index", endChunkIndex, dbErr)
			return dbErr
		}
		dbErr = p.chunkOrm.UpdateBatchHashInRange(p.ctx, startChunkIndex, endChunkIndex, batch.Hash, dbTX)
		if dbErr != nil {
			log.Warn("BatchProposer.UpdateBatchHashInRange update the chunk's batch hash failure", "hash", batch.Hash, "error", dbErr)
			return dbErr
		}
		return nil
	})
	return err
}

func (p *BatchProposer) proposeBatchChunks() ([]*orm.Chunk, error) {
	unbatchedChunkIndex, err := p.batchOrm.GetUnbatchedChunkIndex(p.ctx)
	if err != nil {
		return nil, err
	}

	dbChunks, err := p.chunkOrm.GetChunksFromIndex(p.ctx, unbatchedChunkIndex)
	if err != nil {
		return nil, err
	}

	if len(dbChunks) == 0 {
		return nil, nil
	}

	var totalL1CommitCalldataSize uint32
	var totalL1CommitGas uint64
	var totalChunks uint64
	var totalL1MessagePopped uint64

	parentBatch, err := p.batchOrm.GetLatestBatch(p.ctx)
	if err != nil {
		return nil, err
	}

	// Add extra gas costs
	totalL1CommitGas += 4 * 2100                     // 4 one-time cold sload for commitBatch
	totalL1CommitGas += 20000                        // 1 time sstore
	totalL1CommitGas += types.CalldataNonZeroByteGas // version in calldata

	// adjusting gas:
	// add 1 time cold sload (2100 gas) for L1MessageQueue
	// add 1 time cold address access (2600 gas) for L1MessageQueue
	// minus 1 time warm sload (100 gas) & 1 time warm address access (100 gas)
	totalL1CommitGas += (2100 + 2600 - 100 - 100)
	if parentBatch != nil {
		totalL1CommitGas += types.GetKeccak256Gas(uint64(len(parentBatch.BatchHeader)))         // parent batch header hash
		totalL1CommitGas += types.CalldataNonZeroByteGas * uint64(len(parentBatch.BatchHeader)) // parent batch header in calldata
	}

	for i, chunk := range dbChunks {
		totalL1CommitCalldataSize += chunk.TotalL1CommitCalldataSize
		totalL1CommitGas += chunk.TotalL1CommitGas
		// adjust batch data hash gas cost
		totalL1CommitGas -= types.GetKeccak256Gas(32 * totalChunks)
		totalChunks++
		totalL1CommitGas += types.GetKeccak256Gas(32 * totalChunks)
		// adjust batch header hash gas cost, batch header size: 89 + 32 * ceil(l1MessagePopped / 256)
		totalL1CommitGas -= types.GetKeccak256Gas(89 + 32*(totalL1MessagePopped+255)/256)
		totalL1CommitGas -= types.CalldataNonZeroByteGas * (32 * (totalL1MessagePopped + 255) / 256)
		totalL1MessagePopped += uint64(chunk.TotalL1MessagesPoppedInChunk)
		totalL1CommitGas += types.CalldataNonZeroByteGas * (32 * (totalL1MessagePopped + 255) / 256)
		totalL1CommitGas += types.GetKeccak256Gas(89 + 32*(totalL1MessagePopped+255)/256)
		totalOverEstimateL1CommitGas := uint64(p.gasCostIncreaseMultiplier * float64(totalL1CommitGas))
		if totalChunks > p.maxChunkNumPerBatch ||
			totalL1CommitCalldataSize > p.maxL1CommitCalldataSizePerBatch ||
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
				"currentTotalChunks", totalChunks,
				"maxChunkNumPerBatch", p.maxChunkNumPerBatch,
				"currentL1CommitCalldataSize", totalL1CommitCalldataSize,
				"maxL1CommitCalldataSizePerBatch", p.maxL1CommitCalldataSizePerBatch,
				"currentOverEstimateL1CommitGas", totalOverEstimateL1CommitGas,
				"maxL1CommitGasPerBatch", p.maxL1CommitGasPerBatch)

			p.totalL1CommitGas.Set(float64(totalL1CommitGas))
			p.totalL1CommitCalldataSize.Set(float64(totalL1CommitCalldataSize))
			p.batchChunksNum.Set(float64(len(dbChunks)))
			return dbChunks[:i], nil
		}
	}

	currentTimeSec := uint64(time.Now().Unix())
	if dbChunks[0].StartBlockTime+p.batchTimeoutSec < currentTimeSec {
		log.Warn("first block timeout",
			"start block number", dbChunks[0].StartBlockNumber,
			"first block timestamp", dbChunks[0].StartBlockTime,
			"chunk outdated time threshold", currentTimeSec,
		)
		p.batchFirstBlockTimeoutReached.Inc()
		p.totalL1CommitGas.Set(float64(totalL1CommitGas))
		p.totalL1CommitCalldataSize.Set(float64(totalL1CommitCalldataSize))
		p.batchChunksNum.Set(float64(len(dbChunks)))
		return dbChunks, nil
	}

	log.Debug("pending chunks do not reach one of the constraints or contain a timeout block")
	p.batchChunksProposeNotEnoughTotal.Inc()
	return nil, nil
}

func (p *BatchProposer) dbChunksToBridgeChunks(dbChunks []*orm.Chunk) ([]*types.Chunk, error) {
	chunks := make([]*types.Chunk, len(dbChunks))
	for i, c := range dbChunks {
		wrappedBlocks, err := p.l2BlockOrm.GetL2BlocksInRange(p.ctx, c.StartBlockNumber, c.EndBlockNumber)
		if err != nil {
			log.Error("Failed to fetch wrapped blocks",
				"start number", c.StartBlockNumber, "end number", c.EndBlockNumber, "error", err)
			return nil, err
		}
		chunks[i] = &types.Chunk{
			Blocks: wrappedBlocks,
		}
	}
	return chunks, nil
}
