package l2

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"modernc.org/mathutil"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/sender"
	"scroll-tech/bridge/utils"

	"scroll-tech/database/orm"
)

func (r *Layer2Relayer) checkFinalizingBatches() error {
	var (
		batchLimit              = 10
		blockNumber uint64 = math.MaxUint64
	)
BEGIN:
	batches, err := r.db.GetBlockBatches(
		map[string]interface{}{"rollup_status": orm.RollupFinalizing},
		fmt.Sprintf("AND end_block_number < %d", blockNumber),
		fmt.Sprintf("ORDER BY end_block_number DESC LIMIT %d", batch),
	)
	if err != nil || len(batches) == 0 {
		return err
	}

	for batch := batches[0]; len(batches) > 0; { //nolint:staticcheck
		// If pending txs pool is full, wait a while and retry.
		if r.rollupSender.IsFull() {
			log.Warn("layer2 rollup sender pending finalized tx reaches pending limit")
			time.Sleep(time.Millisecond * 500)
			continue
		}
		batch, batches = batches[0], batches[1:]

		id := batch.ID
		blockNumber = mathutil.MinUint64(blockNumber, batch.EndBlockNumber)

		txStr, err := r.db.GetFinalizeTxHash(id)
		if err != nil {
			log.Error("failed to get finalize_tx_hash from block_batch", "err", err)
			continue
		}

		data, err := r.packFinalizeBatch(id)
		if err != nil {
			log.Error("failed to pack finalize data", "err", err)
			continue
		}

		txID := id + "-finalize"
		err = r.rollupSender.LoadOrSendTx(
			common.HexToHash(txStr.String),
			txID,
			&r.cfg.RollupContractAddress,
			big.NewInt(0),
			data,
		)
		if err != nil {
			log.Error("failed to load or send finalized tx", "batch id", id, "err", err)
		} else {
			r.processingFinalization.Store(txID, id)
		}
	}
	goto BEGIN
}

func (r *Layer2Relayer) packFinalizeBatch(id string) ([]byte, error) {
	proofBuffer, instanceBuffer, err := r.db.GetVerifiedProofAndInstanceByID(id)
	if err != nil {
		log.Warn("fetch get proof by id failed", "id", id, "err", err)
		return nil, err
	}
	if proofBuffer == nil || instanceBuffer == nil {
		log.Warn("proof or instance not ready", "id", id)
		return nil, err
	}
	if len(proofBuffer)%32 != 0 {
		log.Error("proof buffer has wrong length", "id", id, "length", len(proofBuffer))
		return nil, err
	}
	if len(instanceBuffer)%32 != 0 {
		log.Warn("instance buffer has wrong length", "id", id, "length", len(instanceBuffer))
		return nil, err
	}

	proof := utils.BufferToUint256Le(proofBuffer)
	instance := utils.BufferToUint256Le(instanceBuffer)
	data, err := bridge_abi.RollupMetaABI.Pack("finalizeBatchWithProof", common.HexToHash(id), proof, instance)
	if err != nil {
		log.Error("Pack finalizeBatchWithProof failed", "err", err)
		return nil, err
	}
	return data, nil
}

// ProcessCommittedBatches submit proof to layer 1 rollup contract
func (r *Layer2Relayer) ProcessCommittedBatches() {
	// set skipped batches in a single db operation
	if count, err := r.db.UpdateSkippedBatches(); err != nil {
		log.Error("UpdateSkippedBatches failed", "err", err)
		// continue anyway
	} else if count > 0 {
		log.Info("Skipping batches", "count", count)
	}

	// batches are sorted by batch index in increasing order
	batches, err := r.db.GetCommittedBatches(1)
	if err != nil {
		log.Error("Failed to fetch committed L2 batches", "err", err)
		return
	}
	if len(batches) == 0 {
		return
	}
	id := batches[0]
	// @todo add support to relay multiple batches

	status, err := r.db.GetProvingStatusByID(id)
	if err != nil {
		log.Error("GetProvingStatusByID failed", "id", id, "err", err)
		return
	}

	switch status {
	case orm.ProvingTaskUnassigned, orm.ProvingTaskAssigned:
		// The proof for this block is not ready yet.
		return

	case orm.ProvingTaskProved:
		// It's an intermediate state. The roller manager received the proof but has not verified
		// the proof yet. We don't roll up the proof until it's verified.
		return

	case orm.ProvingTaskFailed, orm.ProvingTaskSkipped:
		// note: this is covered by UpdateSkippedBatches, but we keep it for completeness's sake

		if err = r.db.UpdateRollupStatus(r.ctx, id, orm.RollupFinalizationSkipped); err != nil {
			log.Warn("UpdateRollupStatus failed", "id", id, "err", err)
		}

	case orm.ProvingTaskVerified:
		log.Info("Start to roll up zk proof", "id", id)
		success := false

		defer func() {
			// TODO: need to revisit this and have a more fine-grained error handling
			if !success {
				log.Info("Failed to upload the proof, change rollup status to FinalizationSkipped", "id", id)
				if err = r.db.UpdateRollupStatus(r.ctx, id, orm.RollupFinalizationSkipped); err != nil {
					log.Warn("UpdateRollupStatus failed", "id", id, "err", err)
				}
			}
		}()

		// Pack finalize data.
		data, err := r.packFinalizeBatch(id)
		if err != nil {
			return
		}

		txID := id + "-finalize"
		// add suffix `-finalize` to avoid duplication with commit tx in unit tests
		txHash, err := r.rollupSender.SendTransaction(txID, &r.cfg.RollupContractAddress, big.NewInt(0), data)
		hash := &txHash
		if err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) {
				log.Error("finalizeBatchWithProof in layer1 failed", "id", id, "err", err)
			}
			return
		}
		log.Info("finalizeBatchWithProof in layer1", "batch_id", id, "hash", hash)

		// record and sync with db, @todo handle db error
		err = r.db.UpdateFinalizeTxHashAndRollupStatus(r.ctx, id, hash.String(), orm.RollupFinalizing)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "batch_id", id, "err", err)
		}
		success = true
		r.processingFinalization.Store(txID, id)

	default:
		log.Error("encounter unreachable case in ProcessCommittedBatches",
			"block_status", status,
		)
	}
}
