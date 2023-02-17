package l2

import (
	"fmt"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	eth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"
	"scroll-tech/database"

	"scroll-tech/bridge/config"
)

const commitBatchesLimit = 20

type batchProposer struct {
	mutex sync.Mutex

	orm database.OrmFactory

	batchTimeSec        uint64
	batchGasThreshold   uint64
	batchTxNumThreshold uint64
	batchBlocksLimit    uint64

	proofGenerationFreq uint64
	skippedOpcodes      map[string]struct{}
	batchDataBuffer     []*types.BatchData
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
	if len(blocks) == 0 {
		return
	}

	if blocks[0].GasUsed > p.batchGasThreshold {
		log.Warn("gas overflow even for only 1 block", "height", blocks[0].Number, "gas", blocks[0].GasUsed)
		if err = p.createBatchForBlocks(blocks[:1]); err != nil {
			log.Error("failed to create batch", "number", blocks[0].Number, "err", err)
		}
		return
	}

	if blocks[0].TxNum > p.batchTxNumThreshold {
		log.Warn("too many txs even for only 1 block", "height", blocks[0].Number, "tx_num", blocks[0].TxNum)
		if err = p.createBatchForBlocks(blocks[:1]); err != nil {
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

	if err = p.createBatchForBlocks(blocks); err != nil {
		log.Error("failed to create batch", "from", blocks[0].Number, "to", blocks[len(blocks)-1].Number, "err", err)
	}

	p.trySendBatches()
}

func (p *batchProposer) trySendBatches() {
	if len(p.batchDataBuffer) > commitBatchesLimit {
		// err := p.relayer.SendCommitTx(p.batchDataBuffer[0:commitBatchesLimit], p.batchDataBuffer[0:commitBatchesLimit])
		// if err != nil {
		// 	log.Error("SendCommitTx failed", "error", err)
		// 	return err
		// }

		// clear buffer.
		p.batchDataBuffer = p.batchDataBuffer[commitBatchesLimit:]
		//w.batchHashBuffer = w.batchHashBuffer[commitBatchesLimit:]
	}
}

func (p *batchProposer) createBatchForBlocks(blocks []*types.BlockInfo) error {
	batchData, err := p.createBatchData(blocks)
	if err != nil {
		log.Error("createBatchData failed", "error", err)
		return err
	}

	if err := p.addBatchInfoToDB(batchData); err != nil {
		log.Error("addBatchInfoToDB failed", "BatchHash", batchData.Hash(44, common.HexToHash("0")), "error", err) // todo: add real param
		return err
	}

	p.batchDataBuffer = append(p.batchDataBuffer, batchData)

	//p.addBatchInfoToDB(batchData)

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

	// todo: add real param.
	if dbTxErr = p.orm.SetBatchIDForBlocksInDBTx(dbTx, blockIDs, batchData.Hash(44, common.HexToHash("0")).Hex()); dbTxErr != nil {
		return dbTxErr
	}

	dbTxErr = dbTx.Commit()
	return dbTxErr
}

func (p *batchProposer) createBatchData(blocks []*types.BlockInfo) (*types.BatchData, error) {
	var err error

	lastBatch, err := p.orm.GetLatestBatch()
	if err != nil {
		// We should not receive sql.ErrNoRows error. The DB should have the batch entry that contains the genesis block.
		return nil, err
	}

	var traces []*eth_types.BlockTrace
	for _, block := range blocks {
		trs, err := p.orm.GetBlockTraces(map[string]interface{}{"hash": block.Hash})
		if err != nil || len(trs) != 1 {
			log.Error("Failed to GetBlockTraces", "hash", block.Hash, "err", err)
			return nil, err
		}
		traces = append(traces, trs[0])
	}

	return types.NewBatchData(lastBatch, traces), nil
}
