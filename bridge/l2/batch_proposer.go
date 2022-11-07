package l2

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database/orm"
)

// batch-related config
const (
	batchTimeSec      = uint64(5 * 60)  // 5min
	batchGasThreshold = uint64(3000000) // 3M
	batchBlocksLimit  = uint64(100)
)

// TODO:
// + generate batch parallelly
// + TraceHasUnsupportedOpcodes
// + proofGenerationFreq
func (w *WatcherClient) tryProposeBatch() error {
	w.bpMutex.Lock()
	defer w.bpMutex.Unlock()

	blocks, err := w.orm.GetBlockInfos(
		map[string]interface{}{"batch_id": sql.NullString{Valid: false}},
		fmt.Sprintf("order by number DESC LIMIT %d", batchBlocksLimit),
	)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return nil
	}

	idsToBatch := []uint64{}
	blocksToBatch := []*orm.BlockInfo{}
	txNum := uint64(0)
	gasUsed := uint64(0)
	for _, block := range blocks {
		txNum += block.TxNum
		gasUsed += block.GasUsed
		if gasUsed > batchGasThreshold {
			break
		}

		idsToBatch = append(idsToBatch, block.Number)
		blocksToBatch = append(blocksToBatch, block)
	}

	if gasUsed < batchGasThreshold && blocks[0].BlockTimestamp+batchTimeSec < uint64(time.Now().Unix()) {
		return nil
	}

	// keep gasUsed below threshold
	if len(idsToBatch) >= 2 {
		gasUsed -= blocks[len(idsToBatch)-1].GasUsed
		txNum -= blocks[len(idsToBatch)-1].TxNum
		idsToBatch = idsToBatch[:len(idsToBatch)-1]
		blocksToBatch = blocksToBatch[:len(blocksToBatch)-1]
	}

	// TODO: use start_block.parent_hash after we upgrade `BlockTrace` type
	parents, err := w.orm.GetBlockInfos(map[string]interface{}{"numer": idsToBatch[0] - 1})
	if err != nil || len(parents) == 0 {
		return errors.New("Cannot find last batch's end_block")
	}

	return w.createBatchForBlocks(idsToBatch, blocksToBatch, parents[0].Hash, txNum, gasUsed)
}

func (w *WatcherClient) createBatchForBlocks(blockIDs []uint64, blocks []*orm.BlockInfo, parentHash string, txNum uint64, gasUsed uint64) error {
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
	batchID, dbTxErr = w.orm.NewBatchInDBTx(dbTx, startBlock, endBlock, parentHash, txNum, gasUsed)
	if dbTxErr != nil {
		return dbTxErr
	}

	if dbTxErr = w.orm.SetBatchIDForBlocksInDBTx(dbTx, blockIDs, batchID); dbTxErr != nil {
		return dbTxErr
	}

	dbTxErr = dbTx.Commit()
	return dbTxErr
}
