package watcher

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/orm"
)

// chunkRowConsumption is map(sub-circuit name => sub-circuit row count)
type chunkRowConsumption map[string]uint64

// add accumulates row consumption per sub-circuit
func (crc *chunkRowConsumption) add(rowConsumption *gethTypes.RowConsumption) error {
	if rowConsumption == nil {
		return errors.New("rowConsumption is <nil>")
	}
	for _, subCircuit := range *rowConsumption {
		(*crc)[subCircuit.Name] += subCircuit.RowNumber
	}
	return nil
}

// max finds the maximum row consumption among all sub-circuits
func (crc *chunkRowConsumption) max() uint64 {
	var max uint64
	for _, value := range *crc {
		if value > max {
			max = value
		}
	}
	return max
}

// ChunkProposer proposes chunks based on available unchunked blocks.
type ChunkProposer struct {
	ctx context.Context
	db  *gorm.DB

	chunkOrm   *orm.Chunk
	l2BlockOrm *orm.L2Block

	maxTxGasPerChunk                uint64 // temporarily DEPRECATED
	maxL2TxNumPerChunk              uint64
	maxL1CommitGasPerChunk          uint64
	maxL1CommitCalldataSizePerChunk uint64
	maxRowConsumptionPerChunk       uint64
	chunkTimeoutSec                 uint64
	gasCostIncreaseMultiplier       float64

	chunkProposerCircleTotal           prometheus.Counter
	proposeChunkFailureTotal           prometheus.Counter
	proposeChunkUpdateInfoTotal        prometheus.Counter
	proposeChunkUpdateInfoFailureTotal prometheus.Counter
	chunkL2TxNum                       prometheus.Gauge
	chunkEstimateL1CommitGas           prometheus.Gauge
	totalL1CommitCalldataSize          prometheus.Gauge
	totalTxGasUsed                     prometheus.Gauge
	maxTxConsumption                   prometheus.Gauge
	chunkBlocksNum                     prometheus.Gauge
	chunkFirstBlockTimeoutReached      prometheus.Counter
	chunkBlocksProposeNotEnoughTotal   prometheus.Counter
}

// NewChunkProposer creates a new ChunkProposer instance.
func NewChunkProposer(ctx context.Context, cfg *config.ChunkProposerConfig, db *gorm.DB, reg prometheus.Registerer) *ChunkProposer {
	return &ChunkProposer{
		ctx:                             ctx,
		db:                              db,
		chunkOrm:                        orm.NewChunk(db),
		l2BlockOrm:                      orm.NewL2Block(db),
		maxTxGasPerChunk:                cfg.MaxTxGasPerChunk,
		maxL2TxNumPerChunk:              cfg.MaxL2TxNumPerChunk,
		maxL1CommitGasPerChunk:          cfg.MaxL1CommitGasPerChunk,
		maxL1CommitCalldataSizePerChunk: cfg.MaxL1CommitCalldataSizePerChunk,
		maxRowConsumptionPerChunk:       cfg.MaxRowConsumptionPerChunk,
		chunkTimeoutSec:                 cfg.ChunkTimeoutSec,
		gasCostIncreaseMultiplier:       cfg.GasCostIncreaseMultiplier,

		chunkProposerCircleTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_propose_chunk_circle_total",
			Help: "Total number of propose chunk total.",
		}),
		proposeChunkFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_propose_chunk_failure_circle_total",
			Help: "Total number of propose chunk failure total.",
		}),
		proposeChunkUpdateInfoTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_propose_chunk_update_info_total",
			Help: "Total number of propose chunk update info total.",
		}),
		proposeChunkUpdateInfoFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_propose_chunk_update_info_failure_total",
			Help: "Total number of propose chunk update info failure total.",
		}),
		chunkL2TxNum: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "bridge_propose_chunk_l2_tx_num",
			Help: "The chunk l2 tx num",
		}),
		chunkEstimateL1CommitGas: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "bridge_propose_chunk_estimate_l1_commit_gas",
			Help: "The chunk estimate l1 commit gas",
		}),
		totalL1CommitCalldataSize: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "bridge_propose_chunk_total_l1_commit_call_data_size",
			Help: "The total l1 commit call data size",
		}),
		totalTxGasUsed: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "bridge_propose_chunk_total_tx_gas_used",
			Help: "The total tx gas used",
		}),
		maxTxConsumption: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "bridge_propose_chunk_max_tx_consumption",
			Help: "The max tx consumption",
		}),
		chunkBlocksNum: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "bridge_propose_chunk_chunk_block_number",
			Help: "The number of blocks in the chunk",
		}),
		chunkFirstBlockTimeoutReached: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_propose_chunk_first_block_timeout_reached_total",
			Help: "Total times of chunk's first block timeout reached",
		}),
		chunkBlocksProposeNotEnoughTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_propose_chunk_blocks_propose_not_enough_total",
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

func (p *ChunkProposer) updateChunkInfoInDB(chunk *types.Chunk) error {
	if chunk == nil {
		return nil
	}

	p.proposeChunkUpdateInfoTotal.Inc()
	err := p.db.Transaction(func(dbTX *gorm.DB) error {
		dbChunk, err := p.chunkOrm.InsertChunk(p.ctx, chunk, dbTX)
		if err != nil {
			log.Warn("ChunkProposer.InsertChunk failed", "chunk hash", chunk.Hash)
			return err
		}
		if err := p.l2BlockOrm.UpdateChunkHashInRange(p.ctx, dbChunk.StartBlockNumber, dbChunk.EndBlockNumber, dbChunk.Hash, dbTX); err != nil {
			log.Error("failed to update chunk_hash for l2_blocks", "chunk hash", chunk.Hash, "start block", 0, "end block", 0, "err", err)
			return err
		}
		return nil
	})
	return err
}

func (p *ChunkProposer) proposeChunk() (*types.Chunk, error) {
	blocks, err := p.l2BlockOrm.GetUnchunkedBlocks(p.ctx)
	if err != nil {
		return nil, err
	}

	if len(blocks) == 0 {
		return nil, nil
	}

	var chunk types.Chunk
	var totalTxGasUsed uint64
	var totalL2TxNum uint64
	var totalL1CommitCalldataSize uint64
	var totalL1CommitGas uint64
	crc := chunkRowConsumption{}

	for i, block := range blocks {
		chunk.Blocks = append(chunk.Blocks, block)
		totalTxGasUsed += block.Header.GasUsed
		totalL2TxNum += block.L2TxsNum()
		totalL1CommitCalldataSize += block.EstimateL1CommitCalldataSize()
		totalL1CommitGas = chunk.EstimateL1CommitGas()

		if err := crc.add(block.RowConsumption); err != nil {
			return nil, fmt.Errorf("chunk-proposer failed to update chunk row consumption: %v", err)
		}
		crcMax := crc.max()

		if totalL2TxNum > p.maxL2TxNumPerChunk ||
			totalL1CommitCalldataSize > p.maxL1CommitCalldataSizePerChunk ||
			p.gasCostIncreaseMultiplier*float64(totalL1CommitGas) > float64(p.maxL1CommitGasPerChunk) ||
			crcMax > p.maxRowConsumptionPerChunk {
			// Check if the first block breaks hard limits.
			// If so, it indicates there are bugs in sequencer, manual fix is needed.
			if i == 0 {
				if totalL2TxNum > p.maxL2TxNumPerChunk {
					return nil, fmt.Errorf(
						"the first block exceeds l2 tx number limit; block number: %v, number of transactions: %v, max transaction number limit: %v",
						chunk.Blocks[0].Header.Number,
						totalL2TxNum,
						p.maxL2TxNumPerChunk,
					)
				}

				if p.gasCostIncreaseMultiplier*float64(totalL1CommitGas) > float64(p.maxL1CommitGasPerChunk) {
					return nil, fmt.Errorf(
						"the first block exceeds l1 commit gas limit; block number: %v, commit gas: %v, max commit gas limit: %v",
						chunk.Blocks[0].Header.Number,
						totalL1CommitGas,
						p.maxL1CommitGasPerChunk,
					)
				}

				if totalL1CommitCalldataSize > p.maxL1CommitCalldataSizePerChunk {
					return nil, fmt.Errorf(
						"the first block exceeds l1 commit calldata size limit; block number: %v, calldata size: %v, max calldata size limit: %v",
						chunk.Blocks[0].Header.Number,
						totalL1CommitCalldataSize,
						p.maxL1CommitCalldataSizePerChunk,
					)
				}

				if crcMax > p.maxRowConsumptionPerChunk {
					return nil, fmt.Errorf(
						"the first block exceeds row consumption limit; block number: %v, row consumption: %v, max: %v, limit: %v",
						chunk.Blocks[0].Header.Number,
						crc,
						crcMax,
						p.maxRowConsumptionPerChunk,
					)
				}
			}

			p.chunkL2TxNum.Set(float64(totalL2TxNum))
			p.chunkEstimateL1CommitGas.Set(float64(totalL1CommitGas))
			p.totalL1CommitCalldataSize.Set(float64(totalL1CommitCalldataSize))
			p.maxTxConsumption.Set(float64(crcMax))
			p.totalTxGasUsed.Set(float64(totalTxGasUsed))
			p.chunkBlocksNum.Set(float64(len(chunk.Blocks)))
			chunk.Blocks = chunk.Blocks[:len(chunk.Blocks)-1] // remove the last block from chunk
			return &chunk, nil
		}
	}

	currentTimeSec := uint64(time.Now().Unix())
	if blocks[0].Header.Time+p.chunkTimeoutSec < currentTimeSec {
		log.Warn("first block timeout",
			"block number", blocks[0].Header.Number,
			"block timestamp", blocks[0].Header.Time,
			"block outdated time threshold", currentTimeSec,
		)
		p.chunkFirstBlockTimeoutReached.Inc()
		p.chunkL2TxNum.Set(float64(totalL2TxNum))
		p.chunkEstimateL1CommitGas.Set(float64(totalL1CommitGas))
		p.totalL1CommitCalldataSize.Set(float64(totalL1CommitCalldataSize))
		p.maxTxConsumption.Set(float64(crc.max()))
		p.totalTxGasUsed.Set(float64(totalTxGasUsed))
		p.chunkBlocksNum.Set(float64(len(chunk.Blocks)))
		return &chunk, nil
	}

	log.Debug("pending blocks do not reach one of the constraints or contain a timeout block")
	p.chunkBlocksProposeNotEnoughTotal.Inc()
	return nil, nil
}
