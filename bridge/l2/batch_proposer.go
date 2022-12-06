package l2

import (
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database/orm"
)

// batch-related config
const (
	batchTimeSec      = uint64(5 * 60) // 5min
	batchGasThreshold = uint64(3_000_000)
	batchBlocksLimit  = uint64(100)
	maxActiveBatches  = int64(20)
)

// TODO:
// + generate batch parallelly
// + TraceHasUnsupportedOpcodes
// + proofGenerationFreq
func (w *WatcherClient) tryProposeBatch() error {
	w.bpMutex.Lock()
	defer w.bpMutex.Unlock()
	numberOfActiveBatches, err := w.orm.GetNumberOfActiveBatches()
	if err != nil {
		return err
	}
	if numberOfActiveBatches > maxActiveBatches {
		// consider sending error here or not
		return nil
	}

	blocks, err := w.orm.GetUnbatchedBlocks(
		map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", batchBlocksLimit),
	)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return nil
	}

	if blocks[0].GasUsed > batchGasThreshold {
		log.Warn("gas overflow even for only 1 block", "height", blocks[0].Number, "gas", blocks[0].GasUsed)
		return w.createBatchForBlocks(blocks[:1])
	}

	var (
		length  = len(blocks)
		gasUsed uint64
	)
	// add blocks into batch until reach batchGasThreshold
	for i, block := range blocks {
		if gasUsed+block.GasUsed > batchGasThreshold {
			blocks = blocks[:i]
			break
		}
		gasUsed += block.GasUsed
	}

	// if too few gas gathered, but we don't want to halt, we then check the first block in the batch:
	// if it's not old enough we will skip proposing the batch,
	// otherwise we will still propose a batch
	if length == len(blocks) && blocks[0].BlockTimestamp+batchTimeSec > uint64(time.Now().Unix()) {
		return nil
	}

	return w.createBatchForBlocks(blocks)
}

func (w *WatcherClient) createBatchForBlocks(blocks []*orm.BlockInfo) error {
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
		blockIDs       = make([]uint64, len(blocks))
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
