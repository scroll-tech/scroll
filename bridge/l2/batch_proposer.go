package l2

import (
	"fmt"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/bigint"
	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/bridge/config"
)

type batchProposer struct {
	mutex sync.Mutex

	orm database.OrmFactory

	batchTimeSec        uint64
	batchGasThreshold   uint64
	batchTxNumThreshold uint64
	batchBlocksLimit    uint64

	proofGenerationFreq uint64
	skippedOpcodes      map[string]struct{}
}

func newBatchProposer(cfg *config.BatchProposerConfig, orm database.OrmFactory) *batchProposer {
	return &batchProposer{
		mutex:               sync.Mutex{},
		orm:                 orm,
		batchTimeSec:        cfg.BatchTimeSec,
		batchGasThreshold:   cfg.BatchGasThreshold,
		batchTxNumThreshold: cfg.BatchTxNumThreshold,
		batchBlocksLimit:    cfg.BatchBlocksLimit,
		proofGenerationFreq: cfg.ProofGenerationFreq,
		skippedOpcodes:      cfg.SkippedOpcodes,
	}
}

func (w *batchProposer) tryProposeBatch() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	blocks, err := w.orm.GetUnbatchedBlocks(
		map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", w.batchBlocksLimit),
	)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return nil
	}

	if blocks[0].GasUsed > w.batchGasThreshold {
		log.Warn("gas overflow even for only 1 block", "height", blocks[0].Number, "gas", blocks[0].GasUsed)
		return w.createBatchForBlocks(blocks[:1])
	}

	if blocks[0].TxNum > w.batchTxNumThreshold {
		log.Warn("too many txs even for only 1 block", "height", blocks[0].Number, "tx_num", blocks[0].TxNum)
		return w.createBatchForBlocks(blocks[:1])
	}

	var (
		length         = len(blocks)
		gasUsed, txNum uint64
	)
	// add blocks into batch until reach batchGasThreshold
	for i, block := range blocks {
		if (gasUsed+block.GasUsed > w.batchGasThreshold) || (txNum+block.TxNum > w.batchTxNumThreshold) {
			blocks = blocks[:i]
			break
		}
		gasUsed += block.GasUsed
		txNum += block.TxNum
	}

	// if too few gas gathered, but we don't want to halt, we then check the first block in the batch:
	// if it's not old enough we will skip proposing the batch,
	// otherwise we will still propose a batch
	if length == len(blocks) && blocks[0].BlockTimestamp+w.batchTimeSec > uint64(time.Now().Unix()) {
		return nil
	}

	return w.createBatchForBlocks(blocks)
}

func (w *batchProposer) createBatchForBlocks(blocks []*orm.BlockInfo) error {
	dbTx, err := w.orm.Beginx()
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

	var (
		batchID        string
		startBlock     = blocks[0]
		endBlock       = blocks[len(blocks)-1]
		txNum, gasUsed uint64
		blockIDs       = make([]*bigint.BigInt, len(blocks))
	)
	for i, block := range blocks {
		txNum += block.TxNum
		gasUsed += block.GasUsed
		blockIDs[i] = block.Number
	}

	batchID, dbTxErr = w.orm.NewBatchInDBTx(dbTx, startBlock, endBlock, startBlock.ParentHash, txNum, gasUsed)
	if dbTxErr != nil {
		return dbTxErr
	}

	if dbTxErr = w.orm.SetBatchIDForBlocksInDBTx(dbTx, blockIDs, batchID); dbTxErr != nil {
		return dbTxErr
	}

	dbTxErr = dbTx.Commit()
	return dbTxErr
}
