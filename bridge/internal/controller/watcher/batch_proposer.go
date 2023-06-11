package watcher

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
	"gorm.io/gorm"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"

	bridgeAbi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/controller/relayer"
	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
)

var (
	bridgeL2BatchesGasOverThresholdTotalCounter = gethMetrics.NewRegisteredCounter("bridge/l2/batches/gas/over/threshold/total", metrics.ScrollRegistry)
	bridgeL2BatchesTxsOverThresholdTotalCounter = gethMetrics.NewRegisteredCounter("bridge/l2/batches/txs/over/threshold/total", metrics.ScrollRegistry)
	bridgeL2BatchesBlocksCreatedTotalCounter    = gethMetrics.NewRegisteredCounter("bridge/l2/batches/blocks/created/total", metrics.ScrollRegistry)
	bridgeL2BatchesCommitsSentTotalCounter      = gethMetrics.NewRegisteredCounter("bridge/l2/batches/commits/sent/total", metrics.ScrollRegistry)
	bridgeL2BatchesOversizedTotalCounter        = gethMetrics.NewRegisteredCounter("bridge/l2/batches/oversized/total", metrics.ScrollRegistry)
	bridgeL2BatchesTxsCreatedPerBatchGauge      = gethMetrics.NewRegisteredGauge("bridge/l2/batches/txs/created/per/batch", metrics.ScrollRegistry)
	bridgeL2BatchesGasCreatedPerBatchGauge      = gethMetrics.NewRegisteredGauge("bridge/l2/batches/gas/created/per/batch", metrics.ScrollRegistry)
	bridgeL2BatchesPayloadSizePerBatchGauge     = gethMetrics.NewRegisteredGauge("bridge/l2/batches/payload/size/per/batch", metrics.ScrollRegistry)
)

// BatchProposer sends batches commit transactions to relayer.
type BatchProposer struct {
	mutex sync.Mutex
	ctx   context.Context
	db    *gorm.DB

	batchTimeSec             uint64
	batchGasThreshold        uint64
	batchTxNumThreshold      uint64
	batchBlocksLimit         uint64
	batchCommitTimeSec       uint64
	commitCalldataSizeLimit  uint64
	batchDataBufferSizeLimit uint64
	commitCalldataMinSize    uint64
	commitBatchCountLimit    int

	proofGenerationFreq uint64
	batchDataBuffer     []*bridgeTypes.BatchData
	relayer             *relayer.Layer2Relayer

	blockTraceOrm *orm.BlockTrace

	piCfg *bridgeTypes.PublicInputHashConfig
}

// NewBatchProposer will return a new instance of BatchProposer.
func NewBatchProposer(ctx context.Context, cfg *config.BatchProposerConfig, relayer *relayer.Layer2Relayer, db *gorm.DB) *BatchProposer {
	p := &BatchProposer{
		mutex:                    sync.Mutex{},
		ctx:                      ctx,
		db:                       db,
		blockTraceOrm:            orm.NewBlockTrace(db),
		batchTimeSec:             cfg.BatchTimeSec,
		batchGasThreshold:        cfg.BatchGasThreshold,
		batchTxNumThreshold:      cfg.BatchTxNumThreshold,
		batchBlocksLimit:         cfg.BatchBlocksLimit,
		batchCommitTimeSec:       cfg.BatchCommitTimeSec,
		commitCalldataSizeLimit:  cfg.CommitTxCalldataSizeLimit,
		commitCalldataMinSize:    cfg.CommitTxCalldataMinSize,
		commitBatchCountLimit:    int(cfg.CommitTxBatchCountLimit),
		batchDataBufferSizeLimit: 100*cfg.CommitTxCalldataSizeLimit + 1*1024*1024, // @todo: determine the value.
		proofGenerationFreq:      cfg.ProofGenerationFreq,
		piCfg:                    cfg.PublicInputConfig,
		relayer:                  relayer,
	}

	// for graceful restart.
	p.recoverBatchDataBuffer()

	// try to commit the leftover pending batches
	p.TryCommitBatches()

	return p
}

func (p *BatchProposer) recoverBatchDataBuffer() {
	// batches are sorted by batch index in increasing order
	batchHashes, err := p.blockBatchOrm.GetBlockBatchesHashByRollupStatus(types.RollupPending, math.MaxInt32)
	if err != nil {
		log.Crit("Failed to fetch pending L2 batches", "err", err)
	}
	if len(batchHashes) == 0 {
		return
	}
	log.Info("Load pending batches into batchDataBuffer")

	// helper function to cache and get BlockBatch from DB
	blockBatchCache := make(map[string]orm.BlockBatch)
	getBlockBatch := func(batchHash string) (*orm.BlockBatch, error) {
		if blockBatch, ok := blockBatchCache[batchHash]; ok {
			return &blockBatch, nil
		}
		blockBatches, err := p.blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": batchHash}, nil, 0)
		if err != nil || len(blockBatches) == 0 {
			return nil, err
		}
		blockBatchCache[batchHash] = blockBatches[0]
		return &blockBatches[0], nil
	}

	// recover the in-memory batchData from DB
	for _, batchHash := range batchHashes {
		log.Info("recover batch data from pending batch", "batch_hash", batchHash)
		blockBatch, err := getBlockBatch(batchHash)
		if err != nil {
			log.Error("could not get BlockBatch", "batch_hash", batchHash, "error", err)
			continue
		}

		parentBatch, err := getBlockBatch(blockBatch.ParentHash)
		if err != nil {
			log.Error("could not get parent BlockBatch", "batch_hash", batchHash, "error", err)
			continue
		}

		whereFileds := map[string]interface{}{
			"batch_hash": batchHash,
		}
		orderByList := []string{
			"number ASC",
		}

		blockTraces, err := p.blockTraceOrm.GetL2BlockInfos(whereFileds, orderByList, 0)
		if err != nil {
			log.Error("could not GetL2BlockInfos", "batch_hash", batchHash, "error", err)
			continue
		}
		if len(blockTraces) != int(blockBatch.EndBlockNumber-blockBatch.StartBlockNumber+1) {
			log.Error("the number of block info retrieved from DB mistmatches the batch info in the DB",
				"len(blockInfos)", len(blockTraces),
				"expected", blockBatch.EndBlockNumber-blockBatch.StartBlockNumber+1)
			continue
		}

		batchData, err := p.generateBatchData(parentBatch, blockTraces)
		if err != nil {
			continue
		}
		if batchData.Hash().Hex() != batchHash {
			log.Error("the hash from recovered batch data mismatches the DB entry",
				"recovered_batch_hash", batchData.Hash().Hex(),
				"expected", batchHash)
			continue
		}

		p.batchDataBuffer = append(p.batchDataBuffer, batchData)
	}
}

// TryProposeBatch will try to propose a batch.
func (p *BatchProposer) TryProposeBatch() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for p.getBatchDataBufferSize() < p.batchDataBufferSizeLimit {
		orderBy := []string{"number ASC"}
		blockTraces, err := p.blockTraceOrm.GetUnbatchedL2Blocks(map[string]interface{}{}, orderBy, int(p.batchBlocksLimit))
		if err != nil {
			log.Error("failed to get unbatched blocks", "err", err)
			return
		}

		batchCreated := p.proposeBatch(blockTraces)

		// while size of batchDataBuffer < commitCalldataMinSize,
		// proposer keeps fetching and porposing batches.
		if p.getBatchDataBufferSize() >= p.commitCalldataMinSize {
			return
		}

		if !batchCreated {
			// wait for watcher to insert l2 traces.
			time.Sleep(time.Second)
		}
	}
}

// TryCommitBatches will try to commit the pending batches.
func (p *BatchProposer) TryCommitBatches() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.batchDataBuffer) == 0 {
		return
	}

	// estimate the calldata length to determine whether to commit the pending batches
	index := 0
	commit := false
	calldataByteLen := uint64(0)
	for ; index < len(p.batchDataBuffer) && index < p.commitBatchCountLimit; index++ {
		calldataByteLen += bridgeAbi.GetBatchCalldataLength(&p.batchDataBuffer[index].Batch)
		if calldataByteLen > p.commitCalldataSizeLimit {
			commit = true
			if index == 0 {
				log.Warn(
					"The calldata size of one batch is larger than the threshold",
					"batch_hash", p.batchDataBuffer[0].Hash().Hex(),
					"calldata_size", calldataByteLen,
				)
				index = 1
			}
			break
		}
	}
	if !commit && p.batchDataBuffer[0].Timestamp()+p.batchCommitTimeSec > uint64(time.Now().Unix()) {
		return
	}

	// Send commit tx for batchDataBuffer[0:index]
	log.Info("Commit batches", "start_index", p.batchDataBuffer[0].Batch.BatchIndex,
		"end_index", p.batchDataBuffer[index-1].Batch.BatchIndex)
	err := p.relayer.SendCommitTx(p.batchDataBuffer[:index])
	if err != nil {
		// leave the retry to the next ticker
		log.Error("SendCommitTx failed", "error", err)
	} else {
		// pop the processed batches from the buffer
		bridgeL2BatchesCommitsSentTotalCounter.Inc(1)
		p.batchDataBuffer = p.batchDataBuffer[index:]
	}
}

func (p *BatchProposer) proposeBatch(blockTraces []orm.BlockTrace) bool {
	if len(blockTraces) == 0 {
		return false
	}

	approximatePayloadSize := func(hash string) (uint64, error) {
		traces, err := p.blockTraceOrm.GetL2WrappedBlocks(map[string]interface{}{"hash": hash})
		if err != nil {
			return 0, err
		}
		if len(traces) != 1 {
			return 0, fmt.Errorf("unexpected traces length, expected = 1, actual = %d", len(traces))
		}
		size := 0
		for _, tx := range traces[0].Transactions {
			size += len(tx.Data)
		}
		return uint64(size), nil
	}

	firstSize, err := approximatePayloadSize(blockTraces[0].Hash)
	if err != nil {
		log.Error("failed to create batch", "number", blockTraces[0].Number, "err", err)
		return false
	}

	if firstSize > p.commitCalldataSizeLimit {
		log.Warn("oversized payload even for only 1 block", "height", blockTraces[0].Number, "size", firstSize)
		// note: we should probably fail here once we can ensure this will not happen
		if err := p.createBatchForBlocks(blockTraces[:1]); err != nil {
			log.Error("failed to create batch", "number", blockTraces[0].Number, "err", err)
			return false
		}
		bridgeL2BatchesTxsCreatedPerBatchGauge.Update(int64(blockTraces[0].TxNum))
		bridgeL2BatchesGasCreatedPerBatchGauge.Update(int64(blockTraces[0].GasUsed))
		bridgeL2BatchesPayloadSizePerBatchGauge.Update(int64(firstSize))
		bridgeL2BatchesBlocksCreatedTotalCounter.Inc(1)
		bridgeL2BatchesOversizedTotalCounter.Inc(1)
		return true
	}

	if blockTraces[0].GasUsed > p.batchGasThreshold {
		bridgeL2BatchesGasOverThresholdTotalCounter.Inc(1)
		log.Warn("gas overflow even for only 1 block", "height", blockTraces[0].Number, "gas", blockTraces[0].GasUsed)
		if err := p.createBatchForBlocks(blockTraces[:1]); err != nil {
			log.Error("failed to create batch", "number", blockTraces[0].Number, "err", err)
		} else {
			bridgeL2BatchesTxsCreatedPerBatchGauge.Update(int64(blockTraces[0].TxNum))
			bridgeL2BatchesGasCreatedPerBatchGauge.Update(int64(blockTraces[0].GasUsed))
			bridgeL2BatchesPayloadSizePerBatchGauge.Update(int64(firstSize))
			bridgeL2BatchesBlocksCreatedTotalCounter.Inc(1)
		}
		return true
	}

	if blockTraces[0].TxNum > p.batchTxNumThreshold {
		bridgeL2BatchesTxsOverThresholdTotalCounter.Inc(1)
		log.Warn("too many txs even for only 1 block", "height", blockTraces[0].Number, "tx_num", blockTraces[0].TxNum)
		if err := p.createBatchForBlocks(blockTraces[:1]); err != nil {
			log.Error("failed to create batch", "number", blockTraces[0].Number, "err", err)
		} else {
			bridgeL2BatchesTxsCreatedPerBatchGauge.Update(int64(blockTraces[0].TxNum))
			bridgeL2BatchesGasCreatedPerBatchGauge.Update(int64(blockTraces[0].GasUsed))
			bridgeL2BatchesPayloadSizePerBatchGauge.Update(int64(firstSize))
			bridgeL2BatchesBlocksCreatedTotalCounter.Inc(1)
		}
		return true
	}

	var gasUsed, txNum, payloadSize uint64
	reachThreshold := false
	// add blocks into batch until reach batchGasThreshold
	for i, block := range blockTraces {
		size, err := approximatePayloadSize(block.Hash)
		if err != nil {
			log.Error("failed to create batch", "number", block.Number, "err", err)
			return false
		}

		if (gasUsed+block.GasUsed > p.batchGasThreshold) || (txNum+block.TxNum > p.batchTxNumThreshold) || (payloadSize+size > p.commitCalldataSizeLimit) {
			blockTraces = blockTraces[:i]
			reachThreshold = true
			break
		}
		gasUsed += block.GasUsed
		txNum += block.TxNum
		payloadSize += size
	}

	// if too few gas gathered, but we don't want to halt, we then check the first block in the batch:
	// if it's not old enough we will skip proposing the batch,
	// otherwise we will still propose a batch
	if !reachThreshold && blockTraces[0].BlockTimestamp+p.batchTimeSec > uint64(time.Now().Unix()) {
		return false
	}

	if err := p.createBatchForBlocks(blockTraces); err != nil {
		log.Error("failed to create batch", "from", blockTraces[0].Number, "to", blockTraces[len(blockTraces)-1].Number, "err", err)
	} else {
		bridgeL2BatchesTxsCreatedPerBatchGauge.Update(int64(txNum))
		bridgeL2BatchesGasCreatedPerBatchGauge.Update(int64(gasUsed))
		bridgeL2BatchesPayloadSizePerBatchGauge.Update(int64(payloadSize))
		bridgeL2BatchesBlocksCreatedTotalCounter.Inc(int64(len(blockTraces)))
	}

	return true
}

func (p *BatchProposer) createBatchForBlocks(blocks []orm.BlockTrace) error {
	lastBatch, err := p.blockBatchOrm.GetLatestBatch()
	if err != nil {
		// We should not receive sql.ErrNoRows error. The DB should have the batch entry that contains the genesis block.
		return err
	}

	batchData, err := p.generateBatchData(lastBatch, blocks)
	if err != nil {
		log.Error("createBatchData failed", "error", err)
		return err
	}

	if err := orm.AddBatchInfoToDB(p.db, batchData); err != nil {
		log.Error("addBatchInfoToDB failed", "BatchHash", batchData.Hash(), "error", err)
		return err
	}

	p.batchDataBuffer = append(p.batchDataBuffer, batchData)
	return nil
}

func (p *BatchProposer) generateBatchData(parentBatch *orm.BlockBatch, blocks []orm.BlockTrace) (*bridgeTypes.BatchData, error) {
	var wrappedBlocks []*bridgeTypes.WrappedBlock
	for _, block := range blocks {
		trs, err := p.blockTraceOrm.GetL2WrappedBlocks(map[string]interface{}{"hash": block.Hash})
		if err != nil || len(trs) != 1 {
			log.Error("Failed to GetBlockTraces", "hash", block.Hash, "err", err)
			return nil, err
		}

		wrappedBlocks = append(wrappedBlocks, trs[0])
	}

	parentBatchInfo := bridgeTypes.BatchInfo{
		Index:     parentBatch.Index,
		Hash:      parentBatch.Hash,
		StateRoot: parentBatch.StateRoot,
	}
	return bridgeTypes.NewBatchData(&parentBatchInfo, wrappedBlocks, p.piCfg), nil
}

func (p *BatchProposer) getBatchDataBufferSize() (size uint64) {
	for _, batchData := range p.batchDataBuffer {
		size += bridgeAbi.GetBatchCalldataLength(&batchData.Batch)
	}
	return
}
