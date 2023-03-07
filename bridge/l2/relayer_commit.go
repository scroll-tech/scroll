package l2

import (
	"errors"
	"math/big"

	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/sender"
)

// SendCommitTx sends commitBatches tx to L1.
func (r *Layer2Relayer) SendCommitTx(batchData []*types.BatchData) error {
	if len(batchData) == 0 {
		log.Error("SendCommitTx receives empty batch")
		return nil
	}

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
		return err
	}

	// generate a unique txID and send transaction
	var bytes []byte
	for _, batch := range batchData {
		bytes = append(bytes, batch.Hash().Bytes()...)
	}
	txID := crypto.Keccak256Hash(bytes).String()
	txHash, err := r.rollupSender.SendTransaction(txID, &r.cfg.RollupContractAddress, big.NewInt(0), calldata)
	if err != nil {
		if !errors.Is(err, sender.ErrNoAvailableAccount) {
			log.Error("Failed to send commitBatches tx to layer1 ", "err", err)
		}
		return err
	}
	log.Info("Sent the commitBatches tx to layer1",
		"tx_hash", txHash.Hex(),
		"start_batch_index", commitBatches[0].BatchIndex,
		"end_batch_index", commitBatches[len(commitBatches)-1].BatchIndex)

	// record and sync with db, @todo handle db error
	batchHashes := make([]string, len(batchData))
	for i, batch := range batchData {
		batchHashes[i] = batch.Hash().Hex()
		err = r.db.UpdateCommitTxHashAndRollupStatus(r.ctx, batchHashes[i], txHash.String(), types.RollupCommitting)
		if err != nil {
			log.Error("UpdateCommitTxHashAndRollupStatus failed", "hash", batchHashes[i], "index", batch.Batch.BatchIndex, "err", err)
		}
	}
	r.processingBatchesCommitment.Store(txID, batchHashes)
	return nil
}
