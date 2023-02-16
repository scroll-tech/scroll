package l2

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	abi "scroll-tech/bridge/abi"
	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
)

const commitBatchesLimit = 20

//type batchMetaData struct {
//	index  uint64
//	blocks []*orm.BlockInfo
//}

type batchProposer struct {
	mutex sync.Mutex

	orm database.OrmFactory

	batchTimeSec        uint64
	batchGasThreshold   uint64
	batchTxNumThreshold uint64
	batchBlocksLimit    uint64

	proofGenerationFreq uint64
	skippedOpcodes      map[string]struct{}
	batchContextBuffer  []*abi.IScrollChainBatch
	batchHashBuffer     []string
	relayer             *Layer2Relayer
}

func newBatchProposer(cfg *config.BatchProposerConfig, relayer *Layer2Relayer, orm database.OrmFactory) *batchProposer {
	return &batchProposer{
		mutex:               sync.Mutex{},
		orm:                 orm,
		batchTimeSec:        cfg.BatchTimeSec,
		batchGasThreshold:   cfg.BatchGasThreshold,
		batchTxNumThreshold: cfg.BatchTxNumThreshold,
		batchBlocksLimit:    cfg.BatchBlocksLimit,
		proofGenerationFreq: cfg.ProofGenerationFreq,
		skippedOpcodes:      cfg.SkippedOpcodes,
		relayer:             relayer,
	}

	// TODO(colin)
	// graceful restart
	// process unsubmitted batches
}

func (w *batchProposer) tryProposeBatch() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	blocks, err := w.orm.GetUnbatchedBlocks(
		map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", w.batchBlocksLimit),
	)
	if err != nil {
		log.Error("failed to get unbatched blocks", "err", err)
		return
	}
	if len(blocks) == 0 {
		return
	}

	if blocks[0].GasUsed > w.batchGasThreshold {
		log.Warn("gas overflow even for only 1 block", "height", blocks[0].Number, "gas", blocks[0].GasUsed)
		if err = w.createBatchForBlocks(blocks[:1]); err != nil {
			log.Error("failed to create batch", "number", blocks[0].Number, "err", err)
		}
		return
	}

	if blocks[0].TxNum > w.batchTxNumThreshold {
		log.Warn("too many txs even for only 1 block", "height", blocks[0].Number, "tx_num", blocks[0].TxNum)
		if err = w.createBatchForBlocks(blocks[:1]); err != nil {
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
		return
	}

	if err = w.createBatchForBlocks(blocks); err != nil {
		log.Error("failed to create batch", "from", blocks[0].Number, "to", blocks[len(blocks)-1].Number, "err", err)
	}
}

func (w *batchProposer) createBatchForBlocks(blocks []*orm.BlockInfo) error {
	batchContext, err := w.createBridgeBatchData(blocks)
	if err != nil {
		log.Error("createBridgeBatchData failed", "error", err)
		return err
	}

	batchHash, err := w.getBatchHash(blocks)
	if err != nil {
		log.Error("getBatchHash failed", "error", err)
		return err
	}

	w.batchContextBuffer = append(w.batchContextBuffer, batchContext)
	w.batchHashBuffer = append(w.batchHashBuffer, batchHash)
	if len(w.batchContextBuffer) > commitBatchesLimit {
		err := w.relayer.SendCommitTx(w.batchHashBuffer[0:commitBatchesLimit], w.batchContextBuffer[0:commitBatchesLimit])
		if err != nil {
			log.Error("SendCommitTx failed", "error", err)
			return err
		}
		// set batch ID.
		for i := 0; i < commitBatchesLimit; i++ {
			if err := w.updateBlocksInfoInDB(w.batchHashBuffer[i], w.batchContextBuffer[i]); err != nil {
				log.Error("updateBlocksInfoInDB failed", "hash", w.batchHashBuffer[i], "error", err)
				// todo: how to handle db error here.
			}
		}
		// clear buffer.
		w.batchContextBuffer = w.batchContextBuffer[commitBatchesLimit:]
		w.batchHashBuffer = w.batchHashBuffer[commitBatchesLimit:]
	}
	return nil
}

func (w *batchProposer) updateBlocksInfoInDB(batchHash string, layer2Batch *bridge_abi.IScrollChainBatch) error {
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

	var blockIDs = make([]uint64, len(layer2Batch.Blocks))
	for i, block := range layer2Batch.Blocks {
		blockIDs[i] = block.BlockNumber
	}

	if dbTxErr = w.orm.SetBatchIDForBlocksInDBTx(dbTx, blockIDs, batchHash); dbTxErr != nil {
		return dbTxErr
	}

	dbTxErr = dbTx.Commit()
	return dbTxErr
}

func (w *batchProposer) getBatchHash(blocks []*orm.BlockInfo) (string, error) {
	// todo: calculate batch hash, or together in createBridgeBatchData?
	return "bash_hash_ph", nil
}

func (w *batchProposer) createBridgeBatchData(blocks []*orm.BlockInfo) (*abi.IScrollChainBatch, error) {
	var err error

	lastBatch, dbErr := w.orm.GetLatestBatch()
	if dbErr != nil {
		// We should not receive sql.ErrNoRows error. The DB should have the batch entry that contains the genesis block.
		return nil, dbErr
	}

	batchContext := new(abi.IScrollChainBatch)

	// set BatchIndex
	batchContext.BatchIndex = lastBatch.Index + 1

	// set PrevStateRoot
	batchContext.PrevStateRoot, err = newByte32FromString(lastBatch.StateRoot)
	if err != nil {
		log.Error("Corrupted StateRoot in the batch db", "hash", lastBatch.Hash, "index", lastBatch.Index)
		return nil, errors.New("Corrupted data in batch db")
	}

	// set ParentHash
	batchContext.ParentBatchHash, err = newByte32FromString(lastBatch.Hash)
	if err != nil {
		log.Error("Corrupted Hash in the batch db", "hash", lastBatch.Hash, "index", lastBatch.Index)
		return nil, errors.New("Corrupted data in batch db")
	}

	batchContext.Blocks = make([]abi.IScrollChainBlockContext, len(blocks))
	//for i, block := range blocks {
	// blockContext, err = p.createBlockContext(block)
	// batchContext.Blocks[i] = *blockContext
	// if dbErr != nil {
	// 	return nil, dbErr
	// }
	//}
	return batchContext, nil
}

// func (p *batchProposer) createBlockContext(block *orm.BlockInfo) (*abi.IScrollChainBlockContext, error) {

// }

func newByte32FromBytes(b []byte) [32]byte {
	var byte32 [32]byte

	if len(b) > 32 {
		b = b[len(b)-32:]
	}

	copy(byte32[32-len(b):], b)
	return byte32
}

func newByte32FromString(s string) ([32]byte, error) {
	bi, ok := new(big.Int).SetString(s, 10)
	if !ok || len(bi.Bytes()) > 32 {
		var empty [32]byte
		return empty, errors.New("Cannot parse byte32 from string")
	}
	return newByte32FromBytes(bi.Bytes()), nil
}
