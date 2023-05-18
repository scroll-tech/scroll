package orm

import (
	"errors"

	"gorm.io/gorm"

	bridgeTypes "scroll-tech/bridge/internal/types"
)

// AddBatchInfoToDB inserts the batch information to the BlockBatch table and updates the batch_hash
// in all blocks included in the batch.
func AddBatchInfoToDB(db *gorm.DB, batchData *bridgeTypes.BatchData) error {
	blockBatch := NewBlockBatch(db)
	blockTrace := NewBlockTrace(db)
	err := db.Transaction(func(tx *gorm.DB) error {
		rowsAffected, dbTxErr := blockBatch.InsertBlockBatchByBatchData(tx, batchData)
		if dbTxErr != nil {
			return dbTxErr
		}
		if rowsAffected != 1 {
			dbTxErr = errors.New("the InsertBlockBatchByBatchData affected row is not 1")
			return dbTxErr
		}

		var blockIDs = make([]uint64, len(batchData.Batch.Blocks))
		for i, block := range batchData.Batch.Blocks {
			blockIDs[i] = block.BlockNumber
		}

		dbTxErr = blockTrace.UpdateBatchHashForL2Blocks(tx, blockIDs, batchData.Hash().Hex())
		if dbTxErr != nil {
			return dbTxErr
		}
		return nil
	})
	return err
}
