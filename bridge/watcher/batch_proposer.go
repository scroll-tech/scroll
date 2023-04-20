package watcher

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	geth_metrics "github.com/scroll-tech/go-ethereum/metrics"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"

	"scroll-tech/database"

	bridgeabi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/relayer"
)

var (
	bridgeL2BatchesGasOverThresholdTotalCounter = geth_metrics.NewRegisteredCounter("bridge/l2/batches/gas/over/threshold/total", metrics.ScrollRegistry)
	bridgeL2BatchesTxsOverThresholdTotalCounter = geth_metrics.NewRegisteredCounter("bridge/l2/batches/txs/over/threshold/total", metrics.ScrollRegistry)
	bridgeL2BatchesBlocksCreatedTotalCounter    = geth_metrics.NewRegisteredCounter("bridge/l2/batches/blocks/created/total", metrics.ScrollRegistry)
	bridgeL2BatchesCommitsSentTotalCounter      = geth_metrics.NewRegisteredCounter("bridge/l2/batches/commits/sent/total", metrics.ScrollRegistry)

	bridgeL2BatchesTxsCreatedPerBatchGauge = geth_metrics.NewRegisteredGauge("bridge/l2/batches/txs/created/per/batch", metrics.ScrollRegistry)
	bridgeL2BatchesGasCreatedPerBatchGauge = geth_metrics.NewRegisteredGauge("bridge/l2/batches/gas/created/per/batch", metrics.ScrollRegistry)
)

// AddBatchInfoToDB inserts the batch information to the BlockBatch table and updates the batch_hash
// in all blocks included in the batch.
func AddBatchInfoToDB(db database.OrmFactory, batchData *types.BatchData, messages []*types.L2Message, msgProofs [][]byte) error {
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

	for i, msg := range messages {
		if dbTxErr = db.UpdateL2MessageProofInDbTx(context.Background(), dbTx, msg.MsgHash, common.Bytes2Hex(msgProofs[i])); dbTxErr != nil {
			return dbTxErr
		}
	}

	dbTxErr = dbTx.Commit()
	return dbTxErr
}

// BatchProposer sends batches commit transactions to relayer.
type BatchProposer struct {
	mutex sync.Mutex

	ctx context.Context
	orm database.OrmFactory

	batchTimeSec             uint64
	batchGasThreshold        uint64
	batchTxNumThreshold      uint64
	batchBlocksLimit         uint64
	batchCommitTimeSec       uint64
	commitCalldataSizeLimit  uint64
	batchDataBufferSizeLimit uint64
	commitCalldataMinSize    uint64

	proofGenerationFreq uint64
	batchDataBuffer     []*types.BatchData
	relayer             *relayer.Layer2Relayer

	withdrawTrie *WithdrawTrie

	piCfg *types.PublicInputHashConfig
}

// NewBatchProposer will return a new instance of BatchProposer.
func NewBatchProposer(ctx context.Context, cfg *config.BatchProposerConfig, relayer *relayer.Layer2Relayer, orm database.OrmFactory) *BatchProposer {
	withdrawTrie := NewWithdrawTrie()
	p := &BatchProposer{
		mutex:                    sync.Mutex{},
		ctx:                      ctx,
		orm:                      orm,
		batchTimeSec:             cfg.BatchTimeSec,
		batchGasThreshold:        cfg.BatchGasThreshold,
		batchTxNumThreshold:      cfg.BatchTxNumThreshold,
		batchBlocksLimit:         cfg.BatchBlocksLimit,
		batchCommitTimeSec:       cfg.BatchCommitTimeSec,
		commitCalldataSizeLimit:  cfg.CommitTxCalldataSizeLimit,
		commitCalldataMinSize:    cfg.CommitTxCalldataMinSize,
		batchDataBufferSizeLimit: 100*cfg.CommitTxCalldataSizeLimit + 1*1024*1024, // @todo: determine the value.
		proofGenerationFreq:      cfg.ProofGenerationFreq,
		withdrawTrie:             withdrawTrie,
		piCfg:                    cfg.PublicInputConfig,
		relayer:                  relayer,
	}

	// for graceful restart.
	p.recoverBatchDataBuffer()

	// Initialize missing proof before we do anything else
	if err := p.InitializeMissingMessageProof(); err != nil {
		panic(fmt.Sprintf("failed to initialize missing message proof, err: %v", err))
	}

	// try to commit the leftover pending batches
	p.TryCommitBatches()

	return p
}

// InitializeMissingMessageProof will initialize missing message proof.
func (p *BatchProposer) InitializeMissingMessageProof() error {
	firstMsg, err := p.orm.GetL2MessageByNonce(0)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get first l2 message: %v", err)
	}
	// no l2 message
	if firstMsg == nil {
		return nil
	}

	// batch will never be empty, since we always have genesis batch in db
	batch, err := p.orm.GetLatestBatch()
	if err != nil {
		return fmt.Errorf("failed to get latest batch: %v", err)
	}

	var batches []*types.BlockBatch
	batchIndex := batch.Index
	for {
		var nonce sql.NullInt64
		// find last message nonce in before or in this batch
		nonce, err = p.orm.GetLastL2MessageNonceLEHeight(p.ctx, batch.EndBlockNumber)
		if err != nil {
			return fmt.Errorf("failed to last l2 message nonce before %v: %v", batch.EndBlockNumber, err)
		}
		if !nonce.Valid {
			// no message before or in this batch
			break
		}

		var msg *types.L2Message
		msg, err = p.orm.GetL2MessageByNonce(uint64(nonce.Int64))
		if err != nil {
			return fmt.Errorf("failed to l2 message with nonce %v: %v", nonce.Int64, err)
		}
		if msg.Proof.Valid {
			// initialize withdrawTrie
			proofBytes := common.Hex2Bytes(msg.Proof.String)
			p.withdrawTrie.Initialize(uint64(nonce.Int64), common.HexToHash(msg.MsgHash), proofBytes)
			break
		}

		// append unprocessed batch
		batches = append(batches, batch)

		// iterate for next batch
		batchIndex--
		var tBatches []*types.BlockBatch
		tBatches, err = p.orm.GetBlockBatches(map[string]interface{}{
			"index": batchIndex,
		})
		if err != nil {
			return fmt.Errorf("failed to get block batch %v: %v", batchIndex, err)
		}
		if len(tBatches) != 1 {
			return fmt.Errorf("no batch with index %v", batchIndex)
		}
		batch = tBatches[0]
	}

	log.Info("Build withdraw trie with pending messages")
	for i := len(batches) - 1; i >= 0; i-- {
		batch := batches[i]
		msgs, proofs, err := p.appendL2Messages(batch.StartBlockNumber, batch.EndBlockNumber)
		if err != nil {
			return err
		}

		if len(msgs) > 0 {
			dbTx, err := p.orm.Beginx()
			if err != nil {
				return err
			}

			for i, msg := range msgs {
				if dbTxErr := p.orm.UpdateL2MessageProofInDbTx(context.Background(), dbTx, msg.MsgHash, common.Bytes2Hex(proofs[i])); dbTxErr != nil {
					if err := dbTx.Rollback(); err != nil {
						log.Error("dbTx.Rollback()", "err", err)
					}
					return dbTxErr
				}
			}

			if dbTxErr := dbTx.Commit(); dbTxErr != nil {
				if err := dbTx.Rollback(); err != nil {
					log.Error("dbTx.Rollback()", "err", err)
				}
				return dbTxErr
			}
		}
	}
	log.Info("Build withdraw trie finished")

	return nil
}

// appendL2Messages will append all messages between firstBlock and lastBlock (both inclusive) to withdrawTrie and compute corresponding merkle proof of each message.
func (p *BatchProposer) appendL2Messages(firstBlock, lastBlock uint64) ([]*types.L2Message, [][]byte, error) {
	var msgProofs [][]byte
	messages, err := p.orm.GetL2MessagesBetween(
		p.ctx,
		firstBlock,
		lastBlock,
	)
	if err != nil {
		log.Error("GetL2MessagesBetween failed", "error", err)
		return messages, msgProofs, err
	}

	if len(messages) > 0 {
		// double check whether nonce is matched
		if messages[0].Nonce != p.withdrawTrie.NextMessageNonce {
			log.Error("L2 message nonce mismatch", "expected", messages[0].Nonce, "found", p.withdrawTrie.NextMessageNonce)
			return messages, msgProofs, fmt.Errorf("l2 message nonce mismatch, expected: %v, found: %v", messages[0].Nonce, p.withdrawTrie.NextMessageNonce)
		}

		var hashes []common.Hash
		for _, msg := range messages {
			hashes = append(hashes, common.HexToHash(msg.MsgHash))
		}
		msgProofs = p.withdrawTrie.AppendMessages(hashes)
	}
	return messages, msgProofs, nil
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

		blockInfos, err := p.orm.GetL2BlockInfos(
			map[string]interface{}{"batch_hash": batchHash},
			"order by number ASC",
		)

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

// TryProposeBatch will try to propose a batch.
func (p *BatchProposer) TryProposeBatch() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for p.getBatchDataBufferSize() < p.batchDataBufferSizeLimit {
		blocks, err := p.orm.GetUnbatchedL2Blocks(
			map[string]interface{}{},
			fmt.Sprintf("order by number ASC LIMIT %d", p.batchBlocksLimit),
		)
		if err != nil {
			log.Error("failed to get unbatched blocks", "err", err)
			return
		}

		batchCreated := p.ProposeBatch(blocks)

		// while size of batchDataBuffer < commitCalldataMinSize,
		// proposer keeps fetching and porposing batches.
		if p.getBatchDataBufferSize() >= p.commitCalldataMinSize {
			return
		}

		if !batchCreated {
			// wait for watcher to insert l2 traces.
			time.Sleep(time.Second)
		}
	}
}

// TryCommitBatches will try to commit the pending batches.
func (p *BatchProposer) TryCommitBatches() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.batchDataBuffer) == 0 {
		return
	}

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
	if !commit && p.batchDataBuffer[0].Timestamp()+p.batchCommitTimeSec > uint64(time.Now().Unix()) {
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
		bridgeL2BatchesCommitsSentTotalCounter.Inc(1)
		p.batchDataBuffer = p.batchDataBuffer[index:]
	}
}

// ProposeBatch will propose a batch, unit testing only
func (p *BatchProposer) ProposeBatch(blocks []*types.BlockInfo) bool {
	if len(blocks) == 0 {
		return false
	}

	if blocks[0].GasUsed > p.batchGasThreshold {
		bridgeL2BatchesGasOverThresholdTotalCounter.Inc(1)
		log.Warn("gas overflow even for only 1 block", "height", blocks[0].Number, "gas", blocks[0].GasUsed)
		if err := p.createBatchForBlocks(blocks[:1]); err != nil {
			log.Error("failed to create batch", "number", blocks[0].Number, "err", err)
		} else {
			bridgeL2BatchesTxsCreatedPerBatchGauge.Update(int64(blocks[0].TxNum))
			bridgeL2BatchesGasCreatedPerBatchGauge.Update(int64(blocks[0].GasUsed))
			bridgeL2BatchesBlocksCreatedTotalCounter.Inc(1)
		}
		return true
	}

	if blocks[0].TxNum > p.batchTxNumThreshold {
		bridgeL2BatchesTxsOverThresholdTotalCounter.Inc(1)
		log.Warn("too many txs even for only 1 block", "height", blocks[0].Number, "tx_num", blocks[0].TxNum)
		if err := p.createBatchForBlocks(blocks[:1]); err != nil {
			log.Error("failed to create batch", "number", blocks[0].Number, "err", err)
		} else {
			bridgeL2BatchesTxsCreatedPerBatchGauge.Update(int64(blocks[0].TxNum))
			bridgeL2BatchesGasCreatedPerBatchGauge.Update(int64(blocks[0].GasUsed))
			bridgeL2BatchesBlocksCreatedTotalCounter.Inc(1)
		}
		return true
	}

	var gasUsed, txNum uint64
	reachThreshold := false
	// add blocks into batch until reach batchGasThreshold
	for i, block := range blocks {
		if (gasUsed+block.GasUsed > p.batchGasThreshold) || (txNum+block.TxNum > p.batchTxNumThreshold) {
			blocks = blocks[:i]
			reachThreshold = true
			break
		}
		gasUsed += block.GasUsed
		txNum += block.TxNum
	}

	// if too few gas gathered, but we don't want to halt, we then check the first block in the batch:
	// if it's not old enough we will skip proposing the batch,
	// otherwise we will still propose a batch
	if !reachThreshold && blocks[0].BlockTimestamp+p.batchTimeSec > uint64(time.Now().Unix()) {
		return false
	}

	if err := p.createBatchForBlocks(blocks); err != nil {
		log.Error("failed to create batch", "from", blocks[0].Number, "to", blocks[len(blocks)-1].Number, "err", err)
	} else {
		bridgeL2BatchesTxsCreatedPerBatchGauge.Update(int64(txNum))
		bridgeL2BatchesGasCreatedPerBatchGauge.Update(int64(gasUsed))
		bridgeL2BatchesBlocksCreatedTotalCounter.Inc(int64(len(blocks)))
	}

	return true
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

	messages, msgProofs, err := p.appendL2Messages(batchData.Batch.Blocks[0].BlockNumber, batchData.Batch.Blocks[len(batchData.Batch.Blocks)-1].BlockNumber)
	if err != nil {
		return err
	}

	// double check whether message root is matched
	if p.withdrawTrie.MessageRoot() != batchData.Batch.WithdrawTrieRoot {
		log.Error("L2 message root mismatch", "expected", p.withdrawTrie.MessageRoot(), "found", batchData.Batch.WithdrawTrieRoot)
		return fmt.Errorf("l2 message root mismatch, expected: %v, found: %v", p.withdrawTrie.MessageRoot(), batchData.Batch.WithdrawTrieRoot)
	}

	if err := AddBatchInfoToDB(p.orm, batchData, messages, msgProofs); err != nil {
		log.Error("addBatchInfoToDB failed", "BatchHash", batchData.Hash(), "error", err)
		return err
	}

	p.batchDataBuffer = append(p.batchDataBuffer, batchData)
	return nil
}

func (p *BatchProposer) generateBatchData(parentBatch *types.BlockBatch, blocks []*types.BlockInfo) (*types.BatchData, error) {
	var wrappedBlocks []*types.WrappedBlock
	for _, block := range blocks {
		trs, err := p.orm.GetL2WrappedBlocks(map[string]interface{}{"hash": block.Hash})
		if err != nil || len(trs) != 1 {
			log.Error("Failed to GetBlockTraces", "hash", block.Hash, "err", err)
			return nil, err
		}
		wrappedBlocks = append(wrappedBlocks, trs[0])
	}
	return types.NewBatchData(parentBatch, wrappedBlocks, p.piCfg), nil
}

func (p *BatchProposer) getBatchDataBufferSize() (size uint64) {
	for _, batchData := range p.batchDataBuffer {
		size += bridgeabi.GetBatchCalldataLength(&batchData.Batch)
	}
	return
}
