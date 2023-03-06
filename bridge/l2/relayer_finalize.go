package l2

import (
	"errors"
	"fmt"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"math/big"
	"modernc.org/mathutil"
	"scroll-tech/bridge/sender"
	"scroll-tech/bridge/utils"
	"scroll-tech/common/types"
	"time"
)

func (r *Layer2Relayer) checkFinalizingBatches() error {
	var (
		batchLimit = 10
		batchIndex uint64
	)
BEGIN:
	batches, err := r.db.GetBlockBatches(
		map[string]interface{}{"rollup_status": types.RollupFinalizing},
		fmt.Sprintf("AND index > %d", batchIndex),
		fmt.Sprintf("ORDER BY index ASC LIMIT %d", batchLimit),
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

		hash := batch.Hash
		batchIndex = mathutil.MaxUint64(batchIndex, batch.Index)

		txStr, err := r.db.GetFinalizeTxHash(hash)
		if err != nil {
			log.Error("failed to get finalize_tx_hash from block_batch", "err", err)
			continue
		}

		data, err := r.packFinalizeBatch(hash)
		if err != nil {
			log.Error("failed to pack finalize data", "err", err)
			continue
		}

		txID := hash + "-finalize"
		err = r.rollupSender.LoadOrSendTx(
			common.HexToHash(txStr.String),
			txID,
			&r.cfg.RollupContractAddress,
			big.NewInt(0),
			data,
		)
		if err != nil {
			log.Error("failed to load or send finalized tx", "batch hash", hash, "err", err)
		} else {
			r.processingFinalization.Store(txID, hash)
		}
	}
	goto BEGIN
}

func (r *Layer2Relayer) packFinalizeBatch(hash string) ([]byte, error) {
	proofBuffer, instanceBuffer, err := r.db.GetVerifiedProofAndInstanceByHash(hash)
	if err != nil {
		log.Warn("fetch get proof by hash failed", "hash", hash, "err", err)
		return nil, err
	}
	if proofBuffer == nil || instanceBuffer == nil {
		log.Warn("proof or instance not ready", "hash", hash)
		return nil, err
	}
	if len(proofBuffer)%32 != 0 {
		log.Error("proof buffer has wrong length", "hash", hash, "length", len(proofBuffer))
		return nil, err
	}
	if len(instanceBuffer)%32 != 0 {
		log.Warn("instance buffer has wrong length", "hash", hash, "length", len(instanceBuffer))
		return nil, err
	}

	proof := utils.BufferToUint256Le(proofBuffer)
	instance := utils.BufferToUint256Le(instanceBuffer)
	data, err := r.l1RollupABI.Pack("finalizeBatchWithProof", common.HexToHash(hash), proof, instance)
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
	batchHashes, err := r.db.GetCommittedBatches(1)
	if err != nil {
		log.Error("Failed to fetch committed L2 batches", "err", err)
		return
	}
	if len(batchHashes) == 0 {
		return
	}
	hash := batchHashes[0]
	// @todo add support to relay multiple batches

	batches, err := r.db.GetBlockBatches(map[string]interface{}{"hash": hash}, "LIMIT 1")
	if err != nil {
		log.Error("Failed to fetch committed L2 batch", "hash", hash, "err", err)
		return
	}
	if len(batches) == 0 {
		log.Error("Unexpected result for GetBlockBatches", "hash", hash, "len", 0)
		return
	}

	batch := batches[0]
	status := batch.ProvingStatus

	switch status {
	case types.ProvingTaskUnassigned, types.ProvingTaskAssigned:
		// The proof for this block is not ready yet.
		return

	case types.ProvingTaskProved:
		// It's an intermediate state. The roller manager received the proof but has not verified
		// the proof yet. We don't roll up the proof until it's verified.
		return

	case types.ProvingTaskFailed, types.ProvingTaskSkipped:
		// note: this is covered by UpdateSkippedBatches, but we keep it for completeness's sake

		if err = r.db.UpdateRollupStatus(r.ctx, hash, types.RollupFinalizationSkipped); err != nil {
			log.Warn("UpdateRollupStatus failed", "hash", hash, "err", err)
		}

	case types.ProvingTaskVerified:
		log.Info("Start to roll up zk proof", "hash", hash)
		success := false

		previousBatch, err := r.db.GetLatestFinalizingOrFinalizedBatch()

		// skip submitting proof
		if err == nil && uint64(batch.CreatedAt.Sub(*previousBatch.CreatedAt).Seconds()) < r.cfg.FinalizeBatchIntervalSec {
			log.Info(
				"Not enough time passed, skipping",
				"hash", hash,
				"createdAt", batch.CreatedAt,
				"lastFinalizingHash", previousBatch.Hash,
				"lastFinalizingStatus", previousBatch.RollupStatus,
				"lastFinalizingCreatedAt", previousBatch.CreatedAt,
			)

			if err = r.db.UpdateRollupStatus(r.ctx, hash, types.RollupFinalizationSkipped); err != nil {
				log.Warn("UpdateRollupStatus failed", "hash", hash, "err", err)
			} else {
				success = true
			}

			return
		}

		// handle unexpected db error
		if err != nil && err.Error() != "sql: no rows in result set" {
			log.Error("Failed to get latest finalized batch", "err", err)
			return
		}

		defer func() {
			// TODO: need to revisit this and have a more fine-grained error handling
			if !success {
				log.Info("Failed to upload the proof, change rollup status to FinalizationSkipped", "hash", hash)
				if err = r.db.UpdateRollupStatus(r.ctx, hash, types.RollupFinalizationSkipped); err != nil {
					log.Warn("UpdateRollupStatus failed", "hash", hash, "err", err)
				}
			}
		}()

		data, err := r.packFinalizeBatch(hash)
		if err != nil {
			log.Error("Pack finalizeBatchWithProof failed", "err", err)
			return
		}

		txID := hash + "-finalize"
		// add suffix `-finalize` to avoid duplication with commit tx in unit tests
		txHash, err := r.rollupSender.SendTransaction(txID, &r.cfg.RollupContractAddress, big.NewInt(0), data)
		finalizeTxHash := &txHash
		if err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) {
				log.Error("finalizeBatchWithProof in layer1 failed", "hash", hash, "err", err)
			}
			return
		}
		log.Info("finalizeBatchWithProof in layer1", "batch_hash", hash, "tx_hash", hash)

		// record and sync with db, @todo handle db error
		err = r.db.UpdateFinalizeTxHashAndRollupStatus(r.ctx, hash, finalizeTxHash.String(), types.RollupFinalizing)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "batch_hash", hash, "err", err)
		}
		success = true
		r.processingFinalization.Store(txID, hash)

	default:
		log.Error("encounter unreachable case in ProcessCommittedBatches",
			"block_status", status,
		)
	}
}
