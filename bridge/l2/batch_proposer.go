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
	v     *viper.Viper
}

func newBatchProposer(orm database.OrmFactory, v *viper.Viper) *batchProposer {
	return &batchProposer{
		mutex: sync.Mutex{},
		orm:   orm,
		v:     v,
	}
}

func (w *batchProposer) getBatchTimeSec() uint64 {
	return uint64(w.v.GetInt("batch_time_sec"))
}

func (w *batchProposer) getBatchBlocksLimit() uint64 {
	return uint64(w.v.GetInt("batch_block_limit"))
}

func (w *batchProposer) getBatchGasThreshold() uint64 {
	return uint64(w.v.GetInt("batch_gas_threshold"))
}

// nolint:unused
func (w *batchProposer) getProofGenerationFreq() uint64 {
	return uint64(w.v.GetInt("proof_generation_freq"))
}

// nolint:unused
func (w *batchProposer) getSkippedOpcodes() map[string]struct{} {
	skippedOpcodesSlice := w.v.GetStringSlice("skipped_opcodes")
	skippedOpcodes := make(map[string]struct{}, len(skippedOpcodesSlice))
	for _, opcode := range skippedOpcodesSlice {
		skippedOpcodes[opcode] = struct{}{}
	}
	return skippedOpcodes
}

func (w *batchProposer) tryProposeBatch() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	blocks, err := w.orm.GetUnbatchedBlocks(
		map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", w.getBatchBlocksLimit()),
	)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return nil
	}

	if blocks[0].GasUsed > w.getBatchGasThreshold() {
		log.Warn("gas overflow even for only 1 block", "height", blocks[0].Number, "gas", blocks[0].GasUsed)
		return w.createBatchForBlocks(blocks[:1])
	}

	var (
		length  = len(blocks)
		gasUsed uint64
	)
	// add blocks into batch until reach batchGasThreshold
	for i, block := range blocks {
		if gasUsed+block.GasUsed > w.getBatchGasThreshold() {
			blocks = blocks[:i]
			break
		}
		gasUsed += block.GasUsed
	}

	// if too few gas gathered, but we don't want to halt, we then check the first block in the batch:
	// if it's not old enough we will skip proposing the batch,
	// otherwise we will still propose a batch
	if length == len(blocks) && blocks[0].BlockTimestamp+w.getBatchTimeSec() > uint64(time.Now().Unix()) {
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
