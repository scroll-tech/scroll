package l2

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sync"
	"time"

	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"

	"scroll-tech/database"

	bridgeabi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
)

// AddBatchInfoToDB inserts the batch information to the BlockBatch table and updates the batch_hash
// in all blocks included in the batch.
func AddBatchInfoToDB(db database.OrmFactory, batchData *types.BatchData) error {
	dbTx, err := db.Beginx()
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

	if dbTxErr = db.NewBatchInDBTx(dbTx, batchData); dbTxErr != nil {
		return dbTxErr
	}

	var blockIDs = make([]uint64, len(batchData.Batch.Blocks))
	for i, block := range batchData.Batch.Blocks {
		blockIDs[i] = block.BlockNumber
	}

	if dbTxErr = db.SetBatchHashForL2BlocksInDBTx(dbTx, blockIDs, batchData.Hash().Hex()); dbTxErr != nil {
		return dbTxErr
	}

	dbTxErr = dbTx.Commit()
	return dbTxErr
}

type BatchProposer struct {
	mutex sync.Mutex

	ctx context.Context
	orm database.OrmFactory

	batchTimeSec            uint64
	batchGasThreshold       uint64
	batchTxNumThreshold     uint64
	batchBlocksLimit        uint64
	commitCalldataSizeLimit uint64

	proofGenerationFreq uint64
	batchDataBuffer     []*types.BatchData
	relayer             *Layer2Relayer

	piCfg *types.PublicInputHashConfig

	stopCh chan struct{}
}

func NewBatchProposer(ctx context.Context, cfg *config.BatchProposerConfig, relayer *Layer2Relayer, orm database.OrmFactory) *BatchProposer {
	p := &BatchProposer{
		mutex:                   sync.Mutex{},
		ctx:                     ctx,
		orm:                     orm,
		batchTimeSec:            cfg.BatchTimeSec,
		batchGasThreshold:       cfg.BatchGasThreshold,
		batchTxNumThreshold:     cfg.BatchTxNumThreshold,
		batchBlocksLimit:        cfg.BatchBlocksLimit,
		commitCalldataSizeLimit: cfg.CommitTxCalldataSizeLimit,
		proofGenerationFreq:     cfg.ProofGenerationFreq,
		piCfg:                   cfg.PublicInputConfig,
		relayer:                 relayer,
		stopCh:                  make(chan struct{}),
	}

	// for graceful restart.
	p.recoverBatchDataBuffer()

	// try to commit the leftover pending batches
	p.tryCommitBatches()

	return p
}

// Start the Listening process
func (p *BatchProposer) Start() {
	go func() {
		if reflect.ValueOf(p.orm).IsNil() {
			panic("must run BatchProposer with DB")
		}

		ctx, cancel := context.WithCancel(p.ctx)

		// batch proposer loop
		go func(ctx context.Context) {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return

				case <-ticker.C:
					p.tryProposeBatch()
				}
			}
		}(ctx)

		<-p.stopCh
		cancel()
	}()
}

// Stop the Watcher module, for a graceful shutdown.
func (p *BatchProposer) Stop() {
	p.stopCh <- struct{}{}
}

func (p *BatchProposer) recoverBatchDataBuffer() {
	// batches are sorted by batch index in increasing order
	batchHashes, err := p.orm.GetPendingBatches(math.MaxInt32)
	if err != nil {
		log.Crit("Failed to fetch pending L2 batches", "err", err)
	}
	if len(batchHashes) == 0 {
		return
	}
	log.Info("Load pending batches into batchDataBuffer")

	// helper function to cache and get BlockBatch from DB
	blockBatchCache := make(map[string]*types.BlockBatch)
	getBlockBatch := func(batchHash string) (*types.BlockBatch, error) {
		if blockBatch, ok := blockBatchCache[batchHash]; ok {
			return blockBatch, nil
		}
		blockBatches, err := p.orm.GetBlockBatches(map[string]interface{}{"hash": batchHash})
		if err != nil || len(blockBatches) == 0 {
			return nil, err
		}
		blockBatchCache[batchHash] = blockBatches[0]
		return blockBatches[0], nil
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

		blockInfos, err := p.orm.GetL2BlockInfos(map[string]interface{}{"batch_hash": batchHash})
		if err != nil {
			log.Error("could not GetL2BlockInfos", "batch_hash", batchHash, "error", err)
			continue
		}
		if len(blockInfos) != int(blockBatch.EndBlockNumber-blockBatch.StartBlockNumber+1) {
			log.Error("the number of block info retrieved from DB mistmatches the batch info in the DB",
				"len(blockInfos)", len(blockInfos),
				"expected", blockBatch.EndBlockNumber-blockBatch.StartBlockNumber+1)
			continue
		}

		batchData, err := p.generateBatchData(parentBatch, blockInfos)
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

func (p *BatchProposer) tryProposeBatch() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	blocks, err := p.orm.GetUnbatchedL2Blocks(
		map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", p.batchBlocksLimit),
	)
	if err != nil {
		log.Error("failed to get unbatched blocks", "err", err)
		return
	}

	p.proposeBatch(blocks)
	p.tryCommitBatches()
}

func (p *BatchProposer) tryCommitBatches() {
	// estimate the calldata length to determine whether to commit the pending batches
	index := 0
	commit := false
	calldataByteLen := uint64(0)
	for ; index < len(p.batchDataBuffer); index++ {
		calldataByteLen += bridgeabi.GetBatchCalldataLength(&p.batchDataBuffer[index].Batch)
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
	if !commit {
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
		p.batchDataBuffer = p.batchDataBuffer[index:]
	}
}

func (p *BatchProposer) proposeBatch(blocks []*types.BlockInfo) {
	if len(blocks) == 0 {
		return
	}

	if blocks[0].GasUsed > p.batchGasThreshold {
		log.Warn("gas overflow even for only 1 block", "height", blocks[0].Number, "gas", blocks[0].GasUsed)
		if err := p.createBatchForBlocks(blocks[:1]); err != nil {
			log.Error("failed to create batch", "number", blocks[0].Number, "err", err)
		}
		return
	}

	if blocks[0].TxNum > p.batchTxNumThreshold {
		log.Warn("too many txs even for only 1 block", "height", blocks[0].Number, "tx_num", blocks[0].TxNum)
		if err := p.createBatchForBlocks(blocks[:1]); err != nil {
			log.Error("failed to create batch", "number", blocks[0].Number, "err", err)
		}
		return
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
		return
	}

	if err := p.createBatchForBlocks(blocks); err != nil {
		log.Error("failed to create batch", "from", blocks[0].Number, "to", blocks[len(blocks)-1].Number, "err", err)
	}
}

func (p *BatchProposer) createBatchForBlocks(blocks []*types.BlockInfo) error {
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

	if err := AddBatchInfoToDB(p.orm, batchData); err != nil {
		log.Error("addBatchInfoToDB failed", "BatchHash", batchData.Hash(), "error", err)
		return err
	}

	p.batchDataBuffer = append(p.batchDataBuffer, batchData)
	return nil
}

func (p *BatchProposer) generateBatchData(parentBatch *types.BlockBatch, blocks []*types.BlockInfo) (*types.BatchData, error) {
	var traces []*geth_types.BlockTrace
	for _, block := range blocks {
		trs, err := p.orm.GetL2BlockTraces(map[string]interface{}{"hash": block.Hash})
		if err != nil || len(trs) != 1 {
			log.Error("Failed to GetBlockTraces", "hash", block.Hash, "err", err)
			return nil, err
		}
		traces = append(traces, trs[0])
	}
	return types.NewBatchData(parentBatch, traces, p.piCfg), nil
}
