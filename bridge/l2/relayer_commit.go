package l2

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/modern-go/reflect2"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"

	bridge_abi "scroll-tech/bridge/abi"

	"scroll-tech/common/types"
	"scroll-tech/common/utils"

	"scroll-tech/bridge/sender"
)

func (r *Layer2Relayer) checkRollupBatches() error {
	// Check mock generateBatch interface.
	if reflect2.IsNil(r.batchInterface) {
		return nil
	}

	blockBatchCache := make(map[string]*types.BlockBatch)
	getBlockBatch := func(batchHash string) (*types.BlockBatch, error) {
		if blockBatch, ok := blockBatchCache[batchHash]; ok {
			return blockBatch, nil
		}
		blockBatches, err := r.db.GetBlockBatches(map[string]interface{}{"hash": batchHash})
		if err != nil {
			return nil, err
		}
		if len(blockBatches) == 0 {
			return nil, fmt.Errorf("don't has such blockBatch, batchHash: %s", batchHash)
		}
		blockBatchCache[batchHash] = blockBatches[0]
		return blockBatches[0], nil
	}

	var batchIndex uint64
	for {
		blockBatches, err := r.db.GetBlockBatches(
			map[string]interface{}{"rollup_status": types.RollupCommitting},
			fmt.Sprintf("AND commit_tx_hash IN (SELECT commit_tx_hash FROM block_batch WHERE index > %d GROUP BY commit_tx_hash LIMIT 1)", batchIndex),
			fmt.Sprintf("AND index > %d", batchIndex),
			"ORDER BY index ASC",
		)
		if err != nil || len(blockBatches) == 0 {
			return err
		}

		var batchDataBuffer []*types.BatchData
		batchIndex = blockBatches[len(blockBatches)-1].Index
		for _, blockBatch := range blockBatches {
			// Wait until sender's pending is not full.
			utils.TryTimes(-1, func() bool {
				return !r.rollupSender.IsFull()
			})

			var (
				parentBatch *types.BlockBatch
				blockInfos  []*types.BlockInfo
			)
			parentBatch, err = getBlockBatch(blockBatch.ParentHash)
			if err != nil {
				return err
			}
			blockInfos, err = r.db.GetL2BlockInfos(
				map[string]interface{}{"batch_hash": blockBatch.Hash},
				"order by number ASC",
			)
			if err != nil {
				return err
			}
			if len(blockInfos) != int(blockBatch.EndBlockNumber-blockBatch.StartBlockNumber+1) {
				log.Error("the number of block info retrieved from DB mistmatches the blockBatch info in the DB",
					"len(blockInfos)", len(blockInfos),
					"expected", blockBatch.EndBlockNumber-blockBatch.StartBlockNumber+1)
				continue
			}
			var batchData *types.BatchData
			batchData, err = r.GenerateBatchData(parentBatch, blockInfos)
			if err != nil {
				return err
			}
			batchDataBuffer = append(batchDataBuffer, batchData)
		}
		batchHashes, txID, callData, err := r.packBatchData(batchDataBuffer)
		if err != nil {
			return err
		}

		// Handle tx.
		err = r.rollupSender.LoadOrSendTx(
			common.HexToHash(blockBatches[0].CommitTxHash.String),
			txID,
			&r.cfg.RollupContractAddress,
			big.NewInt(0),
			callData,
			0,
		)
		switch true {
		case err == nil:
			r.processingBatchesCommitment.Store(txID, batchHashes)
		case err.Error() == "Batch already commited":
			for _, batchHash := range batchHashes {
				if err = r.db.UpdateRollupStatus(r.ctx, batchHash, types.RollupCommitted); err != nil {
					log.Error("failed to update rollup status when check rollup batched", "batch_hash", batchHash, "err", err)
					return err
				}
			}
		default:
			log.Error("failed to load or send batchData tx")
			return err
		}
	}
}

func (r *Layer2Relayer) packBatchData(batchData []*types.BatchData) ([]string, string, []byte, error) {
	// pack calldata
	commitBatches := make([]bridge_abi.IScrollChainBatch, len(batchData))
	for i, batch := range batchData {
		commitBatches[i] = batch.Batch
	}
	calldata, err := r.l1RollupABI.Pack("commitBatches", commitBatches)
	if err != nil {
		log.Error("Failed to pack commitBatches",
			"error", err,
			"start_batch_index", commitBatches[0].BatchIndex,
			"end_batch_index", commitBatches[len(commitBatches)-1].BatchIndex)
		return nil, "", nil, err
	}

	// generate a unique txID and send transaction
	var (
		bytes       []byte
		batchHashes = make([]string, len(batchData))
	)
	for i, batch := range batchData {
		bytes = append(bytes, batch.Hash().Bytes()...)
		batchHashes[i] = batch.Hash().Hex()
	}
	return batchHashes, crypto.Keccak256Hash(bytes).String(), calldata, nil
}

// SendCommitTx sends commitBatches tx to L1.
func (r *Layer2Relayer) SendCommitTx(batchData []*types.BatchData) error {
	if len(batchData) == 0 {
		log.Error("SendCommitTx receives empty batch")
		return nil
	}

	// pack calldata
	batchHashes, txID, calldata, err := r.packBatchData(batchData)
	if err != nil {
		log.Error("Failed to pack commitBatches",
			"error", err,
			"start_batch_index", batchData[0].Batch.BatchIndex,
			"end_batch_index", batchData[len(batchData)-1].Batch.BatchIndex)
		return err
	}

	txHash, err := r.rollupSender.SendTransaction(txID, &r.cfg.RollupContractAddress, big.NewInt(0), calldata, 0)
	if err != nil {
		if !errors.Is(err, sender.ErrNoAvailableAccount) {
			log.Error("Failed to send commitBatches tx to layer1 ", "err", err)
		}
		return err
	}
	bridgeL2BatchesCommittedTotalCounter.Inc(int64(len(batchHashes)))
	log.Info("Sent the commitBatches tx to layer1",
		"tx_hash", txHash.Hex(),
		"start_batch_index", batchData[0].Batch.BatchIndex,
		"end_batch_index", batchData[len(batchData)-1].Batch.BatchIndex)

	// record and sync with db, @todo handle db error
	for i := range batchData {
		err = r.db.UpdateCommitTxHashAndRollupStatus(r.ctx, batchHashes[i], txHash.String(), types.RollupCommitting)
		if err != nil {
			log.Error("UpdateCommitTxHashAndRollupStatus failed", "hash", batchHashes[i], "index", batchData[i].Batch.BatchIndex, "err", err)
		}
	}
	r.processingBatchesCommitment.Store(txID, batchHashes)
	return nil
}
