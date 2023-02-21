package l2

import (
	"fmt"
	"math"
	"sync"
	"time"

	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"
	"scroll-tech/database"

	bridgeabi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
)

type batchProposer struct {
	mutex sync.Mutex

	orm database.OrmFactory

	batchTimeSec            uint64
	batchGasThreshold       uint64
	batchTxNumThreshold     uint64
	batchBlocksLimit        uint64
	commitCalldataSizeLimit uint64

	proofGenerationFreq uint64
	skippedOpcodes      map[string]struct{}
	batchDataBuffer     []*types.BatchData
	relayer             *Layer2Relayer

	piCfg *types.PublicInputHashConfig
}

func newBatchProposer(cfg *config.BatchProposerConfig, relayer *Layer2Relayer, orm database.OrmFactory) *batchProposer {
	p := &batchProposer{
		mutex:                   sync.Mutex{},
		orm:                     orm,
		batchTimeSec:            cfg.BatchTimeSec,
		batchGasThreshold:       cfg.BatchGasThreshold,
		batchTxNumThreshold:     cfg.BatchTxNumThreshold,
		batchBlocksLimit:        cfg.BatchBlocksLimit,
		commitCalldataSizeLimit: cfg.CommitTxCalldataSizeLimit,
		proofGenerationFreq:     cfg.ProofGenerationFreq,
		skippedOpcodes:          cfg.SkippedOpcodes,
		piCfg:                   cfg.PublicInputConfig,
		relayer:                 relayer,
	}

	// for graceful restart.
	p.recoverBatchDataBuffer()

	return p
}

func (p *batchProposer) recoverBatchDataBuffer() {
	// batches are sorted by batch index in increasing order
	batchesInDB, err := p.orm.GetPendingBatches(math.MaxInt32)
	if err != nil {
		log.Crit("Failed to fetch pending L2 batches", "err", err)
	}
	if len(batchesInDB) == 0 {
		return
	}
	log.Info("Load pending batches into batchDataBuffer")

	// helper function to cache and get BlockBatch from DB
	var blockBatchCache map[string]*types.BlockBatch
	getBlockBatch := func(batchHash string) (*types.BlockBatch, error) {
		if blockBatch, ok := blockBatchCache[batchHash]; ok {
			return blockBatch, nil
		}
		blockBatches, err := p.orm.GetBlockBatches(map[string]interface{}{"hash": batchHash})
		if err != nil {
			return nil, err
		}
		blockBatchCache[batchHash] = blockBatches[0]
		return blockBatches[0], nil
	}

	// recover the in-memory batchData from DB
	for _, batchHash := range batchesInDB {
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

		blockInfos, err := p.orm.GetBlockInfos(map[string]interface{}{"batch_hash": batchHash})
		if err != nil {
			log.Error("could not GetBlockInfos", "batch_hash", batchHash, "error", err)
			continue
		}

		batchData, err := p.generateBatchData(parentBatch, blockInfos)
		if err != nil {
			continue
		}

		p.batchDataBuffer = append(p.batchDataBuffer, batchData)
	}

	// try to commit the leftover pending batches
	p.tryCommitBatches()
}

func (p *batchProposer) tryProposeBatch() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	blocks, err := p.orm.GetUnbatchedBlocks(
		map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", p.batchBlocksLimit),
	)
	if err != nil {
		log.Error("failed to get unbatched blocks", "err", err)
		return
	}

	if numBlocks := p.proposeBatch(blocks); numBlocks > 0 {
		p.tryCommitBatches()
	}
}

func (p *batchProposer) tryCommitBatches() {
	// estimate the calldata length to determine whether to commit the pending batches
	index := 0
	commit := false
	calldataByteLen := uint64(0)
	for ; index < len(p.batchDataBuffer); index++ {
		calldataByteLen += bridgeabi.GetBatchCalldataLength(&p.batchDataBuffer[index].Batch)
		if calldataByteLen > p.commitCalldataSizeLimit {
			commit = true
			if index == 0 {
				log.Warn("The calldata size of the batch is larger than the threshold", "batch_hash", p.batchDataBuffer[index].Hash().Hex(), "calldata_size", calldataByteLen)
			} else {
				index--
			}
			break
		}
	}
	if !commit {
		return
	}

	// try sending commit tx for batchDataBuffer[0:index]
	err := p.relayer.SendCommitTx(p.batchDataBuffer[:index])
	if err != nil {
		// leave the retry to the next ticker
		log.Error("SendCommitTx failed", "error", err)
	} else {
		// pop the processed batches from the buffer
		p.batchDataBuffer = p.batchDataBuffer[index+1:]
	}
}

func (p *batchProposer) proposeBatch(blocks []*types.BlockInfo) uint64 {
	if len(blocks) == 0 {
		return 0
	}

	if blocks[0].GasUsed > p.batchGasThreshold {
		log.Warn("gas overflow even for only 1 block", "height", blocks[0].Number, "gas", blocks[0].GasUsed)
		if err := p.createBatchForBlocks(blocks[:1]); err != nil {
			log.Error("failed to create batch", "number", blocks[0].Number, "err", err)
		}
		return 1
	}

	if blocks[0].TxNum > p.batchTxNumThreshold {
		log.Warn("too many txs even for only 1 block", "height", blocks[0].Number, "tx_num", blocks[0].TxNum)
		if err := p.createBatchForBlocks(blocks[:1]); err != nil {
			log.Error("failed to create batch", "number", blocks[0].Number, "err", err)
		}
		return 1
	}

	var (
		length         = len(blocks)
		gasUsed, txNum uint64
	)
	// add blocks into batch until reach batchGasThreshold
	for i, block := range blocks {
		if (gasUsed+block.GasUsed > p.batchGasThreshold) || (txNum+block.TxNum > p.batchTxNumThreshold) {
			blocks = blocks[:i]
			break
		}
		gasUsed += block.GasUsed
		txNum += block.TxNum
	}

	// if too few gas gathered, but we don't want to halt, we then check the first block in the batch:
	// if it's not old enough we will skip proposing the batch,
	// otherwise we will still propose a batch
	if length == len(blocks) && blocks[0].BlockTimestamp+p.batchTimeSec > uint64(time.Now().Unix()) {
		return 0
	}

	if err := p.createBatchForBlocks(blocks); err != nil {
		log.Error("failed to create batch", "from", blocks[0].Number, "to", blocks[len(blocks)-1].Number, "err", err)
	}
	return uint64(len(blocks))
}

func (p *batchProposer) createBatchForBlocks(blocks []*types.BlockInfo) error {
	lastBatch, err := p.orm.GetLatestBatch()
	if err != nil {
		// We should not receive sql.ErrNoRows error. The DB should have the batch entry that contains the genesis block.
		return err
	}

	batchData, err := p.generateBatchData(lastBatch, blocks)
	if err != nil {
		log.Error("createBatchData failed", "error", err)
		return err
	}

	if err := p.addBatchInfoToDB(batchData); err != nil {
		log.Error("addBatchInfoToDB failed", "BatchHash", batchData.Hash(), "error", err)
		return err
	}

	p.batchDataBuffer = append(p.batchDataBuffer, batchData)
	return nil
}

func (p *batchProposer) addBatchInfoToDB(batchData *types.BatchData) error {
	dbTx, err := p.orm.Beginx()
	if err != nil {
		return err
	}

	var dbTxErr error
	defer func() {
		if dbTxErr != nil {
			if err := dbTx.Rollback(); err != nil {
				log.Error("dbTx.Rollback()", "err", err)
			}
		}
	}()

	if dbTxErr = p.orm.NewBatchInDBTx(dbTx, batchData); dbTxErr != nil {
		return dbTxErr
	}

	var blockIDs = make([]uint64, len(batchData.Batch.Blocks))
	for i, block := range batchData.Batch.Blocks {
		blockIDs[i] = block.BlockNumber
	}

	if dbTxErr = p.orm.SetBatchHashForBlocksInDBTx(dbTx, blockIDs, batchData.Hash().Hex()); dbTxErr != nil {
		return dbTxErr
	}

	dbTxErr = dbTx.Commit()
	return dbTxErr
}

func (p *batchProposer) generateBatchData(parentBatch *types.BlockBatch, blocks []*types.BlockInfo) (*types.BatchData, error) {
	var traces []*geth_types.BlockTrace
	for _, block := range blocks {
		trs, err := p.orm.GetBlockTraces(map[string]interface{}{"hash": block.Hash})
		if err != nil || len(trs) != 1 {
			log.Error("Failed to GetBlockTraces", "hash", block.Hash, "err", err)
			return nil, err
		}
		traces = append(traces, trs[0])
	}

	return types.NewBatchData(parentBatch, traces, p.piCfg), nil
}
