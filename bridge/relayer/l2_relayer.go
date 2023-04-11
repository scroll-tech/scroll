package relayer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"time"

	cmapV2 "github.com/orcaman/concurrent-map/v2"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	geth_metrics "github.com/scroll-tech/go-ethereum/metrics"
	"golang.org/x/sync/errgroup"
	"modernc.org/mathutil"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	cutils "scroll-tech/common/utils"
	"scroll-tech/database"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
	"scroll-tech/bridge/utils"
)

var (
	bridgeL2MsgsRelayedTotalCounter               = geth_metrics.NewRegisteredCounter("bridge/l2/msgs/relayed/total", metrics.ScrollRegistry)
	bridgeL2BatchesFinalizedTotalCounter          = geth_metrics.NewRegisteredCounter("bridge/l2/batches/finalized/total", metrics.ScrollRegistry)
	bridgeL2BatchesCommittedTotalCounter          = geth_metrics.NewRegisteredCounter("bridge/l2/batches/committed/total", metrics.ScrollRegistry)
	bridgeL2MsgsRelayedConfirmedTotalCounter      = geth_metrics.NewRegisteredCounter("bridge/l2/msgs/relayed/confirmed/total", metrics.ScrollRegistry)
	bridgeL2BatchesFinalizedConfirmedTotalCounter = geth_metrics.NewRegisteredCounter("bridge/l2/batches/finalized/confirmed/total", metrics.ScrollRegistry)
	bridgeL2BatchesCommittedConfirmedTotalCounter = geth_metrics.NewRegisteredCounter("bridge/l2/batches/committed/confirmed/total", metrics.ScrollRegistry)
	bridgeL2BatchesSkippedTotalCounter            = geth_metrics.NewRegisteredCounter("bridge/l2/batches/skipped/total", metrics.ScrollRegistry)
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

	db  database.OrmFactory
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
	processingMessage cmapV2.ConcurrentMap[string, string]

	// A list of processing batches commitment.
	// key(string): confirmation ID, value([]string): batch hashes.
	processingBatchesCommitment cmapV2.ConcurrentMap[string, []string]

	// A list of processing batch finalization.
	// key(string): confirmation ID, value(string): batch hash.
	processingFinalization cmapV2.ConcurrentMap[string, string]
}

// NewLayer2Relayer will return a new instance of Layer2RelayerClient
func NewLayer2Relayer(ctx context.Context, l2Client *ethclient.Client, db database.OrmFactory, cfg *config.RelayerConfig) (*Layer2Relayer, error) {
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
		db:  db,

		l2Client: l2Client,

		messageSender:  messageSender,
		l1MessengerABI: bridge_abi.L1ScrollMessengerABI,

		rollupSender: rollupSender,
		l1RollupABI:  bridge_abi.ScrollChainABI,

		gasOracleSender: gasOracleSender,
		l2GasOracleABI:  bridge_abi.L2GasPriceOracleABI,

		minGasLimitForMessageRelay: minGasLimitForMessageRelay,

		minGasPrice:  minGasPrice,
		gasPriceDiff: gasPriceDiff,

		cfg:                         cfg,
		processingMessage:           cmapV2.New[string](),
		processingBatchesCommitment: cmapV2.New[[]string](),
		processingFinalization:      cmapV2.New[string](),
	}
	go layer2Relayer.handleConfirmLoop(ctx)
	return layer2Relayer, nil
}

const processMsgLimit = 100

// CheckSubmittedMessages loads or resends submitted status of txs.
func (r *Layer2Relayer) CheckSubmittedMessages() error {
	var nonce uint64
	for {
		// msgs are sorted by nonce in increasing order
		l2Nonce, msgs, err := r.db.GetL2TxMessages(
			map[string]interface{}{"status": types.MsgSubmitted},
			fmt.Sprintf("AND nonce > %d", nonce),
			fmt.Sprintf("ORDER BY nonce ASC LIMIT %d", processMsgLimit),
		)
		if err != nil {
			log.Error("failed to get l2 submitted messages", "message nonce", nonce, "err", err)
			return err
		}
		if len(msgs) == 0 {
			return nil
		}
		nonce = l2Nonce

		for _, msg := range msgs {
			// TODO: Is it necessary repair tx message?
			if !msg.TxHash.Valid {
				log.Warn("l2 submitted tx message is empty", "tx id", msg.ID)
				continue
			}
			// Wait until sender's pending is not full.
			cutils.TryTimes(-1, func() bool {
				return !r.messageSender.IsFull()
			})

			isResend, tx, err := r.messageSender.LoadOrResendTx(
				msg.GetTxHash(),
				msg.GetSender(),
				msg.GetNonce(),
				msg.ID,
				msg.GetTarget(),
				msg.GetValue(),
				msg.Data,
				r.minGasLimitForMessageRelay,
			)
			if err != nil {
				log.Error("failed to load or send l2 submitted tx", "msg.hash", msg.ID, "err", err)
				return err
			}
			r.processingMessage.Set(msg.ID, msg.ID)
			log.Info("successfully check l2 submitted tx", "resend", isResend, "tx.Hash", tx.Hash().String())
		}
	}
}

// WaitSubmittedMessages wait all the submitted txs are finished.
func (r *Layer2Relayer) WaitSubmittedMessages() {
	for r.processingMessage.Count() != 0 {
		time.Sleep(time.Second)
	}
}

// ProcessSavedEvents relays saved un-processed cross-domain transactions to desired blockchain
func (r *Layer2Relayer) ProcessSavedEvents() {
	batch, err := r.db.GetLatestFinalizedBatch()
	if err != nil {
		log.Error("GetLatestFinalizedBatch failed", "err", err)
		return
	}

	// msgs are sorted by nonce in increasing order
	msgs, err := r.db.GetL2Messages(
		map[string]interface{}{"status": types.MsgPending},
		fmt.Sprintf("AND height<=%d", batch.EndBlockNumber),
		fmt.Sprintf("ORDER BY nonce ASC LIMIT %d", processMsgLimit),
	)

	if err != nil {
		log.Error("Failed to fetch unprocessed L2 messages", "err", err)
		return
	}

	// process messages in batches
	batchSize := mathutil.Min((runtime.GOMAXPROCS(0)+1)/2, r.messageSender.NumberOfAccounts())
	for size := 0; len(msgs) > 0; msgs = msgs[size:] {
		if size = len(msgs); size > batchSize {
			size = batchSize
		}
		var g errgroup.Group
		for _, msg := range msgs[:size] {
			msg := msg
			g.Go(func() error {
				return r.processSavedEvent(msg)
			})
		}
		if err := g.Wait(); err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
				log.Error("failed to process l2 saved event", "err", err)
			}
			return
		}
	}
}

func (r *Layer2Relayer) processSavedEvent(msg *types.L2Message) error {
	// @todo fetch merkle proof from l2geth
	log.Info("Processing L2 Message", "msg.nonce", msg.Nonce, "msg.height", msg.Height)

	// Get the block info that contains the message
	blockInfos, err := r.db.GetL2BlockInfos(map[string]interface{}{"number": msg.Height})
	if err != nil {
		log.Error("Failed to GetL2BlockInfos from DB", "number", msg.Height)
	}
	blockInfo := blockInfos[0]
	if !blockInfo.BatchHash.Valid {
		log.Error("Block has not been batched yet", "number", blockInfo.Number, "msg.nonce", msg.Nonce)
		return nil
	}

	// TODO: rebuild the withdraw trie to generate the merkle proof
	proof := bridge_abi.IL1ScrollMessengerL2MessageProof{
		BatchHash:   common.HexToHash(blockInfo.BatchHash.String),
		MerkleProof: make([]byte, 0),
	}
	from := common.HexToAddress(msg.Sender)
	target := common.HexToAddress(msg.Target)
	value, ok := big.NewInt(0).SetString(msg.Value, 10)
	if !ok {
		// @todo maybe panic?
		log.Error("Failed to parse message value", "msg.nonce", msg.Nonce, "msg.height", msg.Height)
		// TODO: need to skip this message by changing its status to MsgError
	}
	msgNonce := big.NewInt(int64(msg.Nonce))
	calldata := common.Hex2Bytes(msg.Calldata)
	data, err := r.l1MessengerABI.Pack("relayMessageWithProof", from, target, value, msgNonce, calldata, proof)
	if err != nil {
		log.Error("Failed to pack relayMessageWithProof", "msg.nonce", msg.Nonce, "err", err)
		// TODO: need to skip this message by changing its status to MsgError
		return err
	}

	senderAddr, tx, err := r.messageSender.SendTransaction(msg.MsgHash, &r.cfg.MessengerContractAddress, big.NewInt(0), data, r.minGasLimitForMessageRelay)
	if err != nil && err.Error() == "execution reverted: Message expired" {
		return r.db.UpdateLayer2Status(r.ctx, msg.MsgHash, types.MsgExpired)
	}
	if err != nil && err.Error() == "execution reverted: Message was already successfully executed" {
		return r.db.UpdateLayer2Status(r.ctx, msg.MsgHash, types.MsgConfirmed)
	}
	if err != nil {
		if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
			log.Error("Failed to send relayMessageWithProof tx to layer1 ", "msg.height", msg.Height, "msg.MsgHash", msg.MsgHash, "err", err)
		}
		return err
	}
	bridgeL2MsgsRelayedTotalCounter.Inc(1)
	log.Info("relayMessageWithProof to layer1", "msgHash", msg.MsgHash, "txhash", tx.Hash().String())

	// save status in db
	// @todo handle db error
	err = r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, msg.MsgHash, types.MsgSubmitted, tx.Hash().String())
	if err != nil {
		log.Error("UpdateLayer2StatusAndLayer1Hash failed", "msgHash", msg.MsgHash, "err", err)
		return err
	}
	err = r.db.SaveTx(msg.MsgHash, senderAddr.String(), types.L2toL1MessageTx, tx, "")
	if err != nil {
		log.Error("failed to save l2 relay tx message", "msg.hash", msg.MsgHash, "tx.hash", tx.Hash().String(), "err", err)
	}

	r.processingMessage.Set(msg.MsgHash, msg.MsgHash)
	return nil
}

// ProcessGasPriceOracle imports gas price to layer1
func (r *Layer2Relayer) ProcessGasPriceOracle() {
	batch, err := r.db.GetLatestBatch()
	if err != nil {
		log.Error("Failed to GetLatestBatch", "err", err)
		return
	}

	if batch.OracleStatus == types.GasOraclePending {
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

			from, tx, err := r.gasOracleSender.SendTransaction(batch.Hash, &r.cfg.GasPriceOracleContractAddress, big.NewInt(0), data, 0)
			if err != nil {
				if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
					log.Error("Failed to send setL2BaseFee tx to layer2 ", "batch.Hash", batch.Hash, "err", err)
				}
				return
			}

			err = r.db.UpdateL2GasOracleStatusAndOracleTxHash(r.ctx, batch.Hash, types.GasOracleImporting, tx.Hash().String())
			if err != nil {
				log.Error("UpdateGasOracleStatusAndOracleTxHash failed", "batch.Hash", batch.Hash, "err", err)
				return
			}
			// Record gas oracle tx message.
			err = r.db.SaveTx(batch.Hash, from.String(), types.L2toL1GasOracleTx, tx, "")
			if err != nil {
				log.Error("failed to save l2 gas oracle tx message", "batch.Hash", batch.Hash, "tx.Hash", tx.Hash().String(), "err", err)
			}
			r.lastGasPrice = suggestGasPriceUint64
			log.Info("Update l2 gas price", "txHash", tx.Hash().String(), "GasPrice", suggestGasPrice)
		}
	}
}

// CheckRollupCommittingBatches check or resend rollup committing txs.
func (r *Layer2Relayer) CheckRollupCommittingBatches() error {
	txMsgs, err := r.db.GetScrollTxs(
		map[string]interface{}{
			"type":    types.RollUpCommitTx,
			"confirm": false,
		},
		"ORDER BY nonce ASC",
	)
	if err != nil {
		log.Error("failed to get rollupCommitting tx messages", "err", err)
		return err
	}
	if len(txMsgs) == 0 {
		return nil
	}

	for _, msg := range txMsgs {
		if !msg.ExtraData.Valid {
			return fmt.Errorf("batch hash list is empty, tx.id: %s", msg.ID)
		}
		// Wait until sender's pending is not full.
		cutils.TryTimes(-1, func() bool {
			return !r.rollupSender.IsFull()
		})

		isResend, tx, err := r.rollupSender.LoadOrResendTx(
			msg.GetTxHash(),
			msg.GetSender(),
			msg.GetNonce(),
			msg.ID,
			msg.GetTarget(),
			msg.GetValue(),
			msg.Data,
			r.minGasLimitForMessageRelay,
		)
		if err != nil {
			log.Error("failed to load or resend rollup committing tx", "msg.hash", msg.ID, "tx.hash", tx.Hash().String(), "err", err)
			return err
		}
		r.processingBatchesCommitment.Set(msg.ID, strings.Split(msg.ExtraData.String, ","))
		log.Info("successfully resend rollup coimmitting tx", "resend", isResend, "msg.hash", msg.ID, "tx.Hash", tx.Hash().String())
	}
	return nil
}

// WaitRollupCommittingBatches wait all the rollup committing txs are finished.
func (r *Layer2Relayer) WaitRollupCommittingBatches() {
	for r.processingBatchesCommitment.Count() != 0 {
		time.Sleep(time.Second)
	}
}

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
	from, tx, err := r.rollupSender.SendTransaction(txID, &r.cfg.RollupContractAddress, big.NewInt(0), calldata, 0)
	if err != nil {
		if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
			log.Error("Failed to send commitBatches tx to layer1 ", "err", err)
		}
		return err
	}
	bridgeL2BatchesCommittedTotalCounter.Inc(int64(len(commitBatches)))
	log.Info("Sent the commitBatches tx to layer1",
		"tx_hash", tx.Hash().Hex(),
		"start_batch_index", commitBatches[0].BatchIndex,
		"end_batch_index", commitBatches[len(commitBatches)-1].BatchIndex)

	// record and sync with db, @todo handle db error
	batchHashes := make([]string, len(batchData))
	for i, batch := range batchData {
		batchHashes[i] = batch.Hash().Hex()
		err = r.db.UpdateCommitTxHashAndRollupStatus(r.ctx, batchHashes[i], tx.Hash().String(), types.RollupCommitting)
		if err != nil {
			log.Error("UpdateCommitTxHashAndRollupStatus failed", "hash", batchHashes[i], "index", batch.Batch.BatchIndex, "err", err)
		}
	}
	// Record gas oracle tx message.
	err = r.db.SaveTx(txID, from.String(), types.RollUpCommitTx, tx, strings.Join(batchHashes, ","))
	if err != nil {
		log.Error("failed to save l2 commitBatches tx message", "batches.id", txID, "tx.hash", tx.Hash().String(), "err", err)
	}
	r.processingBatchesCommitment.Set(txID, batchHashes)
	return nil
}

// CheckRollupFinalizingBatches rollupStatus: types.RollupCommitting, types.RollupFinalizing
func (r *Layer2Relayer) CheckRollupFinalizingBatches() error {
	var (
		batchIndex uint64
		batchLimit uint64 = 10
	)
	for {
		maxIndex, batches, err := r.db.GetBlockBatchTxMessages(
			map[string]interface{}{"rollup_status": types.RollupFinalizing},
			fmt.Sprintf("AND index > %d", batchIndex),
			fmt.Sprintf("ORDER BY index ASC LIMIT %d", batchLimit),
		)
		if err != nil {
			log.Error("failed to get RollupFinalizing batches", "batch index", batchIndex, "err", err)
			return err
		}
		if len(batches) == 0 {
			return nil
		}
		batchIndex = maxIndex

		for _, msg := range batches {
			// TODO: Is it necessary repair tx message?
			if !msg.TxHash.Valid {
				log.Warn("RollupFinalizing tx message is empty", "tx id", msg.ID)
				continue
			}
			cutils.TryTimes(-1, func() bool {
				return !r.rollupSender.IsFull()
			})

			isResend, tx, err := r.rollupSender.LoadOrResendTx(
				msg.GetTxHash(),
				msg.GetSender(),
				msg.GetNonce(),
				msg.ID,
				msg.GetTarget(),
				msg.GetValue(),
				msg.Data,
				0,
			)
			if err != nil {
				log.Error("failed to load or send rollup finalizing tx", "batch hash", msg.ID, "err", err)
				return err
			}
			r.processingFinalization.Set(msg.ID, msg.ID)
			log.Info("successfully check rollup finalizing tx", "resend", isResend, "tx.Hash", tx.Hash().String())
		}
	}
}

// WaitRollupFinalizingBatches wait all the rollup finalizing txs are finished.
func (r *Layer2Relayer) WaitRollupFinalizingBatches() {
	for r.processingFinalization.Count() != 0 {
		time.Sleep(time.Second)
	}
}

// ProcessCommittedBatches submit proof to layer 1 rollup contract
func (r *Layer2Relayer) ProcessCommittedBatches() {
	// set skipped batches in a single db operation
	if count, err := r.db.UpdateSkippedBatches(); err != nil {
		log.Error("UpdateSkippedBatches failed", "err", err)
		// continue anyway
	} else if count > 0 {
		bridgeL2BatchesSkippedTotalCounter.Inc(count)
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

		proofBuffer, instanceBuffer, err := r.db.GetVerifiedProofAndInstanceByHash(hash)
		if err != nil {
			log.Warn("fetch get proof by hash failed", "hash", hash, "err", err)
			return
		}
		if proofBuffer == nil || instanceBuffer == nil {
			log.Warn("proof or instance not ready", "hash", hash)
			return
		}
		if len(proofBuffer)%32 != 0 {
			log.Error("proof buffer has wrong length", "hash", hash, "length", len(proofBuffer))
			return
		}
		if len(instanceBuffer)%32 != 0 {
			log.Warn("instance buffer has wrong length", "hash", hash, "length", len(instanceBuffer))
			return
		}

		proof := utils.BufferToUint256Le(proofBuffer)
		instance := utils.BufferToUint256Le(instanceBuffer)
		data, err := r.l1RollupABI.Pack("finalizeBatchWithProof", common.HexToHash(hash), proof, instance)
		if err != nil {
			log.Error("Pack finalizeBatchWithProof failed", "err", err)
			return
		}

		txID := hash + "-finalize"
		// add suffix `-finalize` to avoid duplication with commit tx in unit tests
		from, tx, err := r.rollupSender.SendTransaction(txID, &r.cfg.RollupContractAddress, big.NewInt(0), data, 0)
		if err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
				log.Error("finalizeBatchWithProof in layer1 failed", "hash", hash, "err", err)
			}
			return
		}
		bridgeL2BatchesFinalizedTotalCounter.Inc(1)
		log.Info("finalizeBatchWithProof in layer1", "batch_hash", hash, "tx_hash", hash)

		// record and sync with db, @todo handle db error
		err = r.db.UpdateFinalizeTxHashAndRollupStatus(r.ctx, hash, tx.Hash().String(), types.RollupFinalizing)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "batch_hash", hash, "err", err)
		}
		err = r.db.SaveTx(txID, from.String(), types.RollupFinalizeTx, tx, "")
		if err != nil {
			log.Error("failed to save l2 committed tx message", "batch.hash", txID, "tx.hash", tx.Hash().String(), "err", err)
		}

		success = true
		r.processingFinalization.Set(txID, hash)

	default:
		log.Error("encounter unreachable case in ProcessCommittedBatches",
			"block_status", status,
		)
	}
}

func (r *Layer2Relayer) handleConfirmation(confirmation *sender.Confirmation) {
	transactionType := "Unknown"
	// check whether it is message relay transaction
	if msgHash, ok := r.processingMessage.Get(confirmation.ID); ok {
		transactionType = "MessageRelay"
		var status types.MsgStatus
		if confirmation.IsSuccessful {
			status = types.MsgConfirmed
		} else {
			status = types.MsgRelayFailed
			log.Warn("transaction confirmed but failed in layer1", "confirmation", confirmation)
		}
		// @todo handle db error
		err := r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, msgHash, status, confirmation.TxHash.String())
		if err != nil {
			log.Warn("UpdateLayer2StatusAndLayer1Hash failed", "msgHash", msgHash, "err", err)
		}
		if err = r.db.ConfirmTxByID(confirmation.ID, confirmation.TxHash.String()); err != nil {
			log.Warn("failed to delete l2 relayer message tx data", "msg.Hash", confirmation.ID, "tx.Hash", confirmation.TxHash.String(), "err", err)
		}
		bridgeL2MsgsRelayedConfirmedTotalCounter.Inc(1)
		r.processingMessage.Remove(confirmation.ID)
	}

	// check whether it is CommitBatches transaction
	if batchHashes, ok := r.processingBatchesCommitment.Get(confirmation.ID); ok {
		transactionType = "BatchesCommitment"
		var status types.RollupStatus
		if confirmation.IsSuccessful {
			status = types.RollupCommitted
		} else {
			status = types.RollupCommitFailed
			log.Warn("transaction confirmed but failed in layer1", "confirmation", confirmation)
		}
		for _, batchHash := range batchHashes {
			// @todo handle db error
			err := r.db.UpdateCommitTxHashAndRollupStatus(r.ctx, batchHash, confirmation.TxHash.String(), status)
			if err != nil {
				log.Warn("UpdateCommitTxHashAndRollupStatus failed", "batch_hash", batchHash, "err", err)
			}
		}
		if err := r.db.ConfirmTxByID(confirmation.ID, confirmation.TxHash.String()); err != nil {
			log.Warn("failed to delete commitBatches committed tx data", "batched.id", confirmation.ID, "tx.Hash", confirmation.TxHash.String(), "err", err)
		}
		bridgeL2BatchesCommittedConfirmedTotalCounter.Inc(int64(len(batchHashes)))
		r.processingBatchesCommitment.Remove(confirmation.ID)
	}

	// check whether it is proof finalization transaction
	if batchHash, ok := r.processingFinalization.Get(confirmation.ID); ok {
		transactionType = "ProofFinalization"
		var status types.RollupStatus
		if confirmation.IsSuccessful {
			status = types.RollupFinalized
		} else {
			status = types.RollupFinalizeFailed
			log.Warn("transaction confirmed but failed in layer1", "confirmation", confirmation)
		}
		// @todo handle db error
		err := r.db.UpdateFinalizeTxHashAndRollupStatus(r.ctx, batchHash, confirmation.TxHash.String(), status)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "batch_hash", batchHash, "err", err)
		}
		if err = r.db.ConfirmTxByID(confirmation.ID, confirmation.TxHash.String()); err != nil {
			log.Warn("failed to delete finalizeBatchWithProof tx data", "batch.Hash", confirmation.ID, "tx.Hash", confirmation.TxHash.String(), "err", err)
		}
		bridgeL2BatchesFinalizedConfirmedTotalCounter.Inc(1)
		r.processingFinalization.Remove(confirmation.ID)
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
			if !cfm.IsSuccessful {
				// @discuss: maybe make it pending again?
				err := r.db.UpdateL2GasOracleStatusAndOracleTxHash(r.ctx, cfm.ID, types.GasOracleFailed, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateL2GasOracleStatusAndOracleTxHash failed", "err", err)
				}
				log.Warn("transaction confirmed but failed in layer1", "confirmation", cfm)
			} else {
				// @todo handle db error
				err := r.db.UpdateL2GasOracleStatusAndOracleTxHash(r.ctx, cfm.ID, types.GasOracleImported, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateL2GasOracleStatusAndOracleTxHash failed", "err", err)
				}
				if err = r.db.ConfirmTxByID(cfm.ID, cfm.TxHash.String()); err != nil {
					log.Warn("failed to delete l2 gas oracle tx data", "batch.Hash", cfm.ID, "tx.Hash", cfm.TxHash.String(), "err", err)
				}
				log.Info("transaction confirmed in layer1", "confirmation", cfm)
			}
		}
	}
}
