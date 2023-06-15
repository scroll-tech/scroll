package relayer

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
	"gorm.io/gorm"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"

	bridgeAbi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/controller/sender"
	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
)

var (
	bridgeL2MsgsRelayedTotalCounter               = gethMetrics.NewRegisteredCounter("bridge/l2/msgs/relayed/total", metrics.ScrollRegistry)
	bridgeL2BatchesFinalizedTotalCounter          = gethMetrics.NewRegisteredCounter("bridge/l2/batches/finalized/total", metrics.ScrollRegistry)
	bridgeL2BatchesCommittedTotalCounter          = gethMetrics.NewRegisteredCounter("bridge/l2/batches/committed/total", metrics.ScrollRegistry)
	bridgeL2MsgsRelayedConfirmedTotalCounter      = gethMetrics.NewRegisteredCounter("bridge/l2/msgs/relayed/confirmed/total", metrics.ScrollRegistry)
	bridgeL2BatchesFinalizedConfirmedTotalCounter = gethMetrics.NewRegisteredCounter("bridge/l2/batches/finalized/confirmed/total", metrics.ScrollRegistry)
	bridgeL2BatchesCommittedConfirmedTotalCounter = gethMetrics.NewRegisteredCounter("bridge/l2/batches/committed/confirmed/total", metrics.ScrollRegistry)
	bridgeL2BatchesSkippedTotalCounter            = gethMetrics.NewRegisteredCounter("bridge/l2/batches/skipped/total", metrics.ScrollRegistry)
)

// Layer2Relayer is responsible for
//  1. Committing and finalizing L2 blocks on L1
//  2. Relaying messages from L2 to L1
//
// Actions are triggered by new head from layer 1 geth node.
// @todo It's better to be triggered by watcher.
type Layer2Relayer struct {
	ctx context.Context

	l2Client *ethclient.Client

	batchOrm     *orm.Batch
	chunkOrm     *orm.Chunk
	l2BlockOrm   *orm.L2Block
	l2MessageOrm *orm.L2Message

	cfg *config.RelayerConfig

	messageSender  *sender.Sender
	l1MessengerABI *abi.ABI

	rollupSender *sender.Sender
	l1RollupABI  *abi.ABI

	gasOracleSender *sender.Sender
	l2GasOracleABI  *abi.ABI

	minGasLimitForMessageRelay uint64

	lastGasPrice uint64
	minGasPrice  uint64
	gasPriceDiff uint64

	// A list of processing message.
	// key(string): confirmation ID, value(string): layer2 hash.
	processingMessage sync.Map

	// A list of processing batches commitment.
	// key(string): confirmation ID, value(string): batch hash.
	processingBatchCommitment sync.Map

	// A list of processing batch finalization.
	// key(string): confirmation ID, value(string): batch hash.
	processingFinalization sync.Map
}

// NewLayer2Relayer will return a new instance of Layer2RelayerClient
func NewLayer2Relayer(ctx context.Context, l2Client *ethclient.Client, db *gorm.DB, cfg *config.RelayerConfig) (*Layer2Relayer, error) {
	// @todo use different sender for relayer, block commit and proof finalize
	messageSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.MessageSenderPrivateKeys)
	if err != nil {
		log.Error("Failed to create messenger sender", "err", err)
		return nil, err
	}

	rollupSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.RollupSenderPrivateKeys)
	if err != nil {
		log.Error("Failed to create rollup sender", "err", err)
		return nil, err
	}

	gasOracleSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.GasOracleSenderPrivateKeys)
	if err != nil {
		log.Error("Failed to create gas oracle sender", "err", err)
		return nil, err
	}

	var minGasPrice uint64
	var gasPriceDiff uint64
	if cfg.GasOracleConfig != nil {
		minGasPrice = cfg.GasOracleConfig.MinGasPrice
		gasPriceDiff = cfg.GasOracleConfig.GasPriceDiff
	} else {
		minGasPrice = 0
		gasPriceDiff = defaultGasPriceDiff
	}

	minGasLimitForMessageRelay := uint64(defaultL2MessageRelayMinGasLimit)
	if cfg.MessageRelayMinGasLimit != 0 {
		minGasLimitForMessageRelay = cfg.MessageRelayMinGasLimit
	}

	layer2Relayer := &Layer2Relayer{
		ctx: ctx,

		batchOrm:     orm.NewBatch(db),
		l2MessageOrm: orm.NewL2Message(db),
		l2BlockOrm:   orm.NewL2Block(db),

		l2Client: l2Client,

		messageSender:  messageSender,
		l1MessengerABI: bridgeAbi.L1ScrollMessengerABI,

		rollupSender: rollupSender,
		l1RollupABI:  bridgeAbi.ScrollChainABI,

		gasOracleSender: gasOracleSender,
		l2GasOracleABI:  bridgeAbi.L2GasPriceOracleABI,

		minGasLimitForMessageRelay: minGasLimitForMessageRelay,

		minGasPrice:  minGasPrice,
		gasPriceDiff: gasPriceDiff,

		cfg:                       cfg,
		processingMessage:         sync.Map{},
		processingBatchCommitment: sync.Map{},
		processingFinalization:    sync.Map{},
	}
	go layer2Relayer.handleConfirmLoop(ctx)
	return layer2Relayer, nil
}

// ProcessGasPriceOracle imports gas price to layer1
func (r *Layer2Relayer) ProcessGasPriceOracle() {
	batch, err := r.batchOrm.GetLatestBatch(r.ctx)
	if err != nil {
		log.Error("Failed to GetLatestBatch", "err", err)
		return
	}

	if types.GasOracleStatus(batch.OracleStatus) == types.GasOraclePending {
		suggestGasPrice, err := r.l2Client.SuggestGasPrice(r.ctx)
		if err != nil {
			log.Error("Failed to fetch SuggestGasPrice from l2geth", "err", err)
			return
		}
		suggestGasPriceUint64 := uint64(suggestGasPrice.Int64())
		expectedDelta := r.lastGasPrice * r.gasPriceDiff / gasPriceDiffPrecision

		// last is undefine or (suggestGasPriceUint64 >= minGasPrice && exceed diff)
		if r.lastGasPrice == 0 || (suggestGasPriceUint64 >= r.minGasPrice && (suggestGasPriceUint64 >= r.lastGasPrice+expectedDelta || suggestGasPriceUint64 <= r.lastGasPrice-expectedDelta)) {
			data, err := r.l2GasOracleABI.Pack("setL2BaseFee", suggestGasPrice)
			if err != nil {
				log.Error("Failed to pack setL2BaseFee", "batch.Hash", batch.Hash, "GasPrice", suggestGasPrice.Uint64(), "err", err)
				return
			}

			hash, err := r.gasOracleSender.SendTransaction(batch.Hash, &r.cfg.GasPriceOracleContractAddress, big.NewInt(0), data, 0)
			if err != nil {
				if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
					log.Error("Failed to send setL2BaseFee tx to layer2 ", "batch.Hash", batch.Hash, "err", err)
				}
				return
			}

			updateFields := map[string]interface{}{
				"oracle_status":  types.GasOracleImporting,
				"oracle_tx_hash": hash.String(),
			}
			if err := r.batchOrm.UpdateBatch(r.ctx, batch.Hash, updateFields); err != nil {
				log.Error("Failed to update gas oracle status and oracle tx hash", "batch.Hash", batch.Hash, "err", err)
				return
			}
			r.lastGasPrice = suggestGasPriceUint64
			log.Info("Update l2 gas price", "txHash", hash.String(), "GasPrice", suggestGasPrice)
		}
	}
}

func (r *Layer2Relayer) ProcessPendingBatches() {
	// get pending batch from database
	pendingBatches, err := r.batchOrm.GetPendingBatches(r.ctx, 10)
	if err != nil || len(pendingBatches) == 0 {
		log.Error("Failed to fetch pending L2 batches", "err", err)
		return
	}
	for _, batch := range pendingBatches {

		// get current header and parent header.
		currentBatchHeader, err := r.batchOrm.GetBatchHeader(r.ctx, batch.Index, r.chunkOrm, r.l2BlockOrm)
		if err != nil {
			log.Error("Failed to get batch header", "error", err)
			return
		}
		// batch.Index > 0 since we have imported genesis batch.
		parentBatchHeader, err := r.batchOrm.GetBatchHeader(r.ctx, batch.Index-1, r.chunkOrm, r.l2BlockOrm)
		if err != nil {
			log.Error("Failed to get parent batch header", "error", err)
			return
		}

		// get the chunks for the batch
		startChunkIndex := batch.StartChunkIndex
		endChunkIndex := batch.EndChunkIndex
		dbChunks, err := r.chunkOrm.RangeGetChunks(r.ctx, startChunkIndex, endChunkIndex)
		if err != nil {
			log.Error("Failed to fetch chunks", "error", err)
			return
		}

		encodedChunks := make([][]byte, len(dbChunks))
		for i, c := range dbChunks {
			wrappedBlocks, err := r.l2BlockOrm.RangeGetL2Blocks(r.ctx, c.StartBlockNumber, c.EndBlockNumber)
			if err != nil {
				log.Error("Failed to fetch wrapped blocks", "error", err)
				return
			}
			chunk := &bridgeTypes.Chunk{
				Blocks: wrappedBlocks,
			}
			chunkBytes, err := chunk.Encode()
			if err != nil {
				log.Error("Failed to encode chunk", "error", err)
				return
			}
			encodedChunks[i] = chunkBytes
		}

		calldata, err := r.l1RollupABI.Pack("commitBatch", currentBatchHeader.Version(), parentBatchHeader.Encode(), encodedChunks, currentBatchHeader.EncodeSkippedL1MessageBitmap())
		if err != nil {
			log.Error("Failed to pack commitBatch", "batch_index", batch.Index, "error", err)
			return
		}

		// send transaction
		txID := batch.Hash + "-commit"
		txHash, err := r.rollupSender.SendTransaction(txID, &r.cfg.RollupContractAddress, big.NewInt(0), calldata, 0)
		if err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
				log.Error("Failed to send commitBatch tx to layer1 ", "err", err)
			}
			return
		}

		updateFields := map[string]interface{}{
			"commit_tx_hash": txHash.String(),
			"rollup_status":  types.RollupCommitting,
		}
		if err := r.batchOrm.UpdateBatch(r.ctx, batch.Hash, updateFields); err != nil {
			log.Error("UpdateBatch failed", "hash", batch.Hash, "index", batch.Index, "err", err)
			return
		}
		bridgeL2BatchesCommittedTotalCounter.Inc(1)
		r.processingBatchCommitment.Store(txID, batch.Hash)
		log.Info("Sent the commitBatch tx to layer1", "tx_hash", txHash.Hex(), "batch_index", batch.Index)
	}
}

// ProcessCommittedBatches submit proof to layer 1 rollup contract
func (r *Layer2Relayer) ProcessCommittedBatches() {
	// set skipped batches in a single db operation
	if count, err := r.batchOrm.UpdateSkippedBatches(r.ctx); err != nil {
		log.Error("UpdateSkippedBatches failed", "err", err)
		// continue anyway
	} else if count > 0 {
		bridgeL2BatchesSkippedTotalCounter.Inc(int64(count))
		log.Info("Skipping batches", "count", count)
	}

	fields := map[string]interface{}{
		"rollup_status": types.RollupCommitted,
	}
	orderByList := []string{"index ASC"}
	limit := 1
	batches, err := r.batchOrm.GetBatches(r.ctx, fields, orderByList, limit)
	if err != nil {
		log.Error("Failed to fetch committed L2 batches", "err", err)
		return
	}
	if len(batches) == 0 {
		log.Error("Unexpected result for GetBlockBatches", "len", 0)
		return
	}

	batch := batches[0]
	hash := batch.Hash
	status := types.ProvingStatus(batch.ProvingStatus)
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
		updateFields := map[string]interface{}{
			"rollup_status": types.RollupFinalizationSkipped,
		}
		if err = r.batchOrm.UpdateBatch(r.ctx, hash, updateFields); err != nil {
			log.Warn("UpdateBatch failed", "hash", hash, "err", err)
		}
	case types.ProvingTaskVerified:
		log.Info("Start to roll up zk proof", "hash", hash)
		success := false

		rollupStatues := []types.RollupStatus{
			types.RollupFinalizing,
			types.RollupFinalized,
		}
		previousBatch, err := r.batchOrm.GetLatestBatchByRollupStatus(rollupStatues)
		// skip submitting proof
		if err == nil && uint64(batch.CreatedAt.Sub(previousBatch.CreatedAt).Seconds()) < r.cfg.FinalizeBatchIntervalSec {
			log.Info(
				"Not enough time passed, skipping",
				"hash", hash,
				"createdAt", batch.CreatedAt,
				"lastFinalizingHash", previousBatch.Hash,
				"lastFinalizingStatus", previousBatch.RollupStatus,
				"lastFinalizingCreatedAt", previousBatch.CreatedAt,
			)

			updateFields := map[string]interface{}{
				"rollup_status": types.RollupFinalizationSkipped,
			}
			if err = r.batchOrm.UpdateBatch(r.ctx, hash, updateFields); err != nil {
				log.Warn("UpdateBatch failed", "hash", hash, "err", err)
			} else {
				success = true
			}

			return
		}

		// handle unexpected db error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("Failed to get latest finalized batch", "err", err)
			return
		}

		defer func() {
			// TODO: need to revisit this and have a more fine-grained error handling
			if !success {
				log.Info("Failed to upload the proof, change rollup status to FinalizationSkipped", "hash", hash)
				updateFields := map[string]interface{}{
					"rollup_status": types.RollupFinalizationSkipped,
				}
				if err = r.batchOrm.UpdateBatch(r.ctx, hash, updateFields); err != nil {
					log.Warn("UpdateBatch failed", "hash", hash, "err", err)
				}
			}
		}()

		aggProof, err := r.batchOrm.GetVerifiedProofByHash(r.ctx, hash)
		if err != nil {
			log.Warn("get verified proof by hash failed", "hash", hash, "err", err)
			return
		}

		if err = aggProof.SanityCheck(); err != nil {
			log.Warn("agg_proof sanity check fails", "hash", hash, "error", err)
			return
		}

		batchHeader, err := r.batchOrm.GetBatchHeader(r.ctx, batch.Index, r.chunkOrm, r.l2BlockOrm)
		if err != nil {
			log.Error("GetBatchHeader failed", "index", batch.Index, "err", err)
			return
		}

		data, err := r.l1RollupABI.Pack(
			"finalizeBatchWithProof",
			batchHeader,
			common.HexToHash(previousBatch.StateRoot),
			common.HexToHash(batch.StateRoot),
			common.HexToHash(batch.WithdrawRoot),
			aggProof.Proof,
		)
		if err != nil {
			log.Error("Pack finalizeBatchWithProof failed", "err", err)
			return
		}

		txID := hash + "-finalize"
		// add suffix `-finalize` to avoid duplication with commit tx in unit tests
		txHash, err := r.rollupSender.SendTransaction(txID, &r.cfg.RollupContractAddress, big.NewInt(0), data, 0)
		finalizeTxHash := &txHash
		if err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
				log.Error("finalizeBatchWithProof in layer1 failed", "hash", hash, "err", err)
			}
			return
		}
		bridgeL2BatchesFinalizedTotalCounter.Inc(1)
		log.Info("finalizeBatchWithProof in layer1", "batch_hash", hash, "tx_hash", hash)

		// record and sync with db, @todo handle db error
		updateFields := map[string]interface{}{
			"finalize_tx_hash": finalizeTxHash.String(),
			"rollup_status":    types.RollupFinalizing,
		}
		if err = r.batchOrm.UpdateBatch(r.ctx, hash, updateFields); err != nil {
			log.Warn("UpdateBatch failed", "batch_hash", hash, "err", err)
		}
		success = true
		r.processingFinalization.Store(txID, hash)

	default:
		log.Error("encounter unreachable case in ProcessCommittedBatches",
			"block_status", status,
		)
	}
}

func (r *Layer2Relayer) handleConfirmation(confirmation *sender.Confirmation) {
	transactionType := "Unknown"
	// check whether it is message relay transaction
	if msgHash, ok := r.processingMessage.Load(confirmation.ID); ok {
		transactionType = "MessageRelay"
		var status types.MsgStatus
		if confirmation.IsSuccessful {
			status = types.MsgConfirmed
		} else {
			status = types.MsgRelayFailed
			log.Warn("transaction confirmed but failed in layer1", "confirmation", confirmation)
		}

		updateFields := map[string]interface{}{
			"commit_tx_hash": confirmation.TxHash.String(),
			"rollup_status":  status,
		}
		err := r.batchOrm.UpdateBatch(r.ctx, msgHash.(string), updateFields)
		if err != nil {
			log.Warn("UpdateBatch failed", "msgHash", msgHash.(string), "err", err)
		}
		bridgeL2MsgsRelayedConfirmedTotalCounter.Inc(1)
		r.processingMessage.Delete(confirmation.ID)
	}

	// check whether it is CommitBatches transaction
	if batchBatches, ok := r.processingBatchCommitment.Load(confirmation.ID); ok {
		transactionType = "BatchesCommitment"
		batchHashes := batchBatches.([]string)
		var status types.RollupStatus
		if confirmation.IsSuccessful {
			status = types.RollupCommitted
		} else {
			status = types.RollupCommitFailed
			log.Warn("transaction confirmed but failed in layer1", "confirmation", confirmation)
		}
		for _, batchHash := range batchHashes {
			updateFields := map[string]interface{}{
				"finalize_tx_hash": confirmation.TxHash.String(),
				"rollup_status":    status,
			}
			if err := r.batchOrm.UpdateBatch(r.ctx, batchHash, updateFields); err != nil {
				log.Warn("UpdateBatch failed", "batch_hash", batchHash, "err", err)
			}
		}
		bridgeL2BatchesCommittedConfirmedTotalCounter.Inc(int64(len(batchHashes)))
		r.processingBatchCommitment.Delete(confirmation.ID)
	}

	// check whether it is proof finalization transaction
	if batchHash, ok := r.processingFinalization.Load(confirmation.ID); ok {
		transactionType = "ProofFinalization"
		var status types.RollupStatus
		if confirmation.IsSuccessful {
			status = types.RollupFinalized
		} else {
			status = types.RollupFinalizeFailed
			log.Warn("transaction confirmed but failed in layer1", "confirmation", confirmation)
		}

		updateFields := map[string]interface{}{
			"finalize_tx_hash": confirmation.TxHash.String(),
			"rollup_status":    status,
		}
		if err := r.batchOrm.UpdateBatch(r.ctx, batchHash.(string), updateFields); err != nil {
			log.Warn("UpdateBatch failed", "batch_hash", batchHash.(string), "err", err)
		}
		bridgeL2BatchesFinalizedConfirmedTotalCounter.Inc(1)
		r.processingFinalization.Delete(confirmation.ID)
	}
	log.Info("transaction confirmed in layer1", "type", transactionType, "confirmation", confirmation)
}

func (r *Layer2Relayer) handleConfirmLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case confirmation := <-r.messageSender.ConfirmChan():
			r.handleConfirmation(confirmation)
		case confirmation := <-r.rollupSender.ConfirmChan():
			r.handleConfirmation(confirmation)
		case cfm := <-r.gasOracleSender.ConfirmChan():
			updateFields := map[string]interface{}{
				"oracle_tx_hash": cfm.TxHash.String(),
			}
			if !cfm.IsSuccessful {
				// @discuss: maybe make it pending again?
				updateFields["oracle_status"] = types.GasOracleFailed
				if err := r.batchOrm.UpdateBatch(r.ctx, cfm.ID, updateFields); err != nil {
					log.Warn("UpdateBatch failed", "err", err)
				}
				log.Warn("transaction confirmed but failed in layer1", "confirmation", cfm)
			} else {
				// @todo handle db error
				updateFields["oracle_status"] = types.GasOracleImported
				if err := r.batchOrm.UpdateBatch(r.ctx, cfm.ID, updateFields); err != nil {
					log.Warn("UpdateBatch failed", "err", err)
				}
				log.Info("transaction confirmed in layer1", "confirmation", cfm)
			}
		}
	}
}
