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
)

// TODO:
// + generate batch parallelly
// + TraceHasUnsupportedOpcodes
// + proofGenerationFreq
func (w *WatcherClient) tryProposeBatch() error {
	w.bpMutex.Lock()
	defer w.bpMutex.Unlock()

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

	idsToBatch := []uint64{}
	var txNum uint64
	var gasUsed uint64
	// add blocks into batch until reach batchGasThreshold
	for _, block := range blocks {
		if gasUsed+block.GasUsed > batchGasThreshold {
			break
		}
		txNum += block.TxNum
		gasUsed += block.GasUsed
		idsToBatch = append(idsToBatch, block.Number)
	}

	// if too few gas gathered, but we don't want to halt, we then check the first block in the batch:
	// if it's not old enough we will skip proposing the batch,
	// otherwise we will still propose a batch
	if len(idsToBatch) == len(blocks) && gasUsed < batchGasThreshold &&
		blocks[0].BlockTimestamp+batchTimeSec > uint64(time.Now().Unix()) {
		return nil
	}

	if len(idsToBatch) == 0 {
		log.Warn("gas overflow even for only 1 block", "gas", blocks[0].GasUsed)
		txNum = blocks[0].TxNum
		gasUsed = blocks[0].GasUsed
		idsToBatch = []uint64{blocks[0].Number}
	}

	return w.createBatchForBlocks(idsToBatch, blocks[:len(idsToBatch)], txNum, gasUsed)
}

func (w *WatcherClient) createBatchForBlocks(blockIDs []uint64, blocks []*orm.BlockInfo, txNum uint64, gasUsed uint64) error {
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

	startBlock := blocks[0]
	endBlock := blocks[len(blocks)-1]
	var batchID string
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
