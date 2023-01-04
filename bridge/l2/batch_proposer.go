package l2

import (
	"fmt"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/viper"
	"scroll-tech/database"
	"scroll-tech/database/orm"
)

type batchProposer struct {
	mutex sync.Mutex
	orm   database.OrmFactory
	vp    *viper.Viper
}

func newBatchProposer(vp *viper.Viper, orm database.OrmFactory) *batchProposer {
	return &batchProposer{
		mutex: sync.Mutex{},
		orm:   orm,
		vp:    vp,
	}
}

func (w *batchProposer) tryProposeBatch() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	batchBlocksLimit := w.vp.GetUint64("batch_blocks_limit")
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

	batchGasThreshold := w.vp.GetUint64("batch_gas_threshold")
	if blocks[0].GasUsed > batchGasThreshold {
		log.Warn("gas overflow even for only 1 block", "height", blocks[0].Number, "gas", blocks[0].GasUsed)
		return w.createBatchForBlocks(blocks[:1])
	}

	batchTxNumThreshold := w.vp.GetUint64("batch_tx_num_threshold")
	if blocks[0].TxNum > batchTxNumThreshold {
		log.Warn("too many txs even for only 1 block", "height", blocks[0].Number, "tx_num", blocks[0].TxNum)
		return w.createBatchForBlocks(blocks[:1])
	}

	var (
		length         = len(blocks)
		gasUsed, txNum uint64
	)
	// add blocks into batch until reach batchGasThreshold
	for i, block := range blocks {
		if (gasUsed+block.GasUsed > batchGasThreshold) || (txNum+block.TxNum > batchTxNumThreshold) {
			blocks = blocks[:i]
			break
		}
		gasUsed += block.GasUsed
		txNum += block.TxNum
	}

	// if too few gas gathered, but we don't want to halt, we then check the first block in the batch:
	// if it's not old enough we will skip proposing the batch,
	// otherwise we will still propose a batch
	batchTimeSec := w.vp.GetUint64("batch_time_sec")
	if length == len(blocks) && blocks[0].BlockTimestamp+batchTimeSec > uint64(time.Now().Unix()) {
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
