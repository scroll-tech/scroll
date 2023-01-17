package l2

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	// not sure if this will make problems when relay with l1geth

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"golang.org/x/sync/errgroup"
	"modernc.org/mathutil"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
	"scroll-tech/bridge/utils"
)

// Layer2Relayer is responsible for
//  1. Committing and finalizing L2 blocks on L1
//  2. Relaying messages from L2 to L1
//
// Actions are triggered by new head from layer 1 geth node.
// @todo It's better to be triggered by watcher.
type Layer2Relayer struct {
	ctx context.Context

	db  database.OrmFactory
	cfg *config.RelayerConfig

	messageSender  *sender.Sender
	messageCh      <-chan *sender.Confirmation
	l1MessengerABI *abi.ABI

	rollupSender *sender.Sender
	rollupCh     <-chan *sender.Confirmation
	l1RollupABI  *abi.ABI

	// A list of processing message.
	// key(string): confirmation ID, value(string): layer2 hash.
	processingMessage sync.Map

	// A list of processing batch commitment.
	// key(string): confirmation ID, value(string): batch id.
	processingCommitment sync.Map

	// A list of processing batch finalization.
	// key(string): confirmation ID, value(string): batch id.
	processingFinalization sync.Map

	stopCh chan struct{}
}

// NewLayer2Relayer will return a new instance of Layer2RelayerClient
func NewLayer2Relayer(ctx context.Context, db database.OrmFactory, cfg *config.RelayerConfig) (*Layer2Relayer, error) {
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

	return &Layer2Relayer{
		ctx:                    ctx,
		db:                     db,
		messageSender:          messageSender,
		messageCh:              messageSender.ConfirmChan(),
		l1MessengerABI:         bridge_abi.L1MessengerMetaABI,
		rollupSender:           rollupSender,
		rollupCh:               rollupSender.ConfirmChan(),
		l1RollupABI:            bridge_abi.RollupMetaABI,
		cfg:                    cfg,
		processingMessage:      sync.Map{},
		processingCommitment:   sync.Map{},
		processingFinalization: sync.Map{},
		stopCh:                 make(chan struct{}),
	}, nil
}

const processMsgLimit = 100

// ProcessSavedEvents relays saved un-processed cross-domain transactions to desired blockchain
func (r *Layer2Relayer) ProcessSavedEvents(wg *sync.WaitGroup) {
	defer wg.Done()
	batch, err := r.db.GetLatestFinalizedBatch()
	if err != nil {
		log.Error("GetLatestFinalizedBatch failed", "err", err)
		return
	}

	// msgs are sorted by nonce in increasing order
	msgs, err := r.db.GetL2Messages(
		map[string]interface{}{"status": orm.MsgPending},
		fmt.Sprintf("AND height<=%d ORDER BY nonce ASC LIMIT %d", batch.EndBlockNumber, processMsgLimit),
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
				return r.processSavedEvent(msg, batch.Index)
			})
		}
		if err := g.Wait(); err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) {
				log.Error("failed to process l2 saved event", "err", err)
			}
			return
		}
	}
}

func (r *Layer2Relayer) processSavedEvent(msg *orm.L2Message, index uint64) error {
	// @todo fetch merkle proof from l2geth
	log.Info("Processing L2 Message", "msg.nonce", msg.Nonce, "msg.height", msg.Height)

	proof := bridge_abi.IL1ScrollMessengerL2MessageProof{
		BlockHeight: big.NewInt(int64(msg.Height)),
		BatchIndex:  big.NewInt(0).SetUint64(index),
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
	fee, _ := big.NewInt(0).SetString(msg.Fee, 10)
	deadline := big.NewInt(int64(msg.Deadline))
	msgNonce := big.NewInt(int64(msg.Nonce))
	calldata := common.Hex2Bytes(msg.Calldata)
	data, err := r.l1MessengerABI.Pack("relayMessageWithProof", from, target, value, fee, deadline, msgNonce, calldata, proof)
	if err != nil {
		log.Error("Failed to pack relayMessageWithProof", "msg.nonce", msg.Nonce, "err", err)
		// TODO: need to skip this message by changing its status to MsgError
		return err
	}

	hash, err := r.messageSender.SendTransaction(msg.MsgHash, &r.cfg.MessengerContractAddress, big.NewInt(0), data)
	if err != nil {
		if !errors.Is(err, sender.ErrNoAvailableAccount) {
			log.Error("Failed to send relayMessageWithProof tx to layer1 ", "msg.height", msg.Height, "msg.MsgHash", msg.MsgHash, "err", err)
		}
		return err
	}
	log.Info("relayMessageWithProof to layer1", "msgHash", msg.MsgHash, "txhash", hash.String())

	// save status in db
	// @todo handle db error
	err = r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, msg.MsgHash, orm.MsgSubmitted, hash.String())
	if err != nil {
		log.Error("UpdateLayer2StatusAndLayer1Hash failed", "msgHash", msg.MsgHash, "err", err)
		return err
	}
	r.processingMessage.Store(msg.MsgHash, msg.MsgHash)
	return nil
}

// ProcessPendingBatches submit batch data to layer 1 rollup contract
func (r *Layer2Relayer) ProcessPendingBatches(wg *sync.WaitGroup) {
	defer wg.Done()
	// batches are sorted by batch index in increasing order
	batchesInDB, err := r.db.GetPendingBatches(1)
	if err != nil {
		log.Error("Failed to fetch pending L2 batches", "err", err)
		return
	}
	if len(batchesInDB) == 0 {
		return
	}
	id := batchesInDB[0]
	// @todo add support to relay multiple batches

	batches, err := r.db.GetBlockBatches(map[string]interface{}{"id": id})
	if err != nil || len(batches) == 0 {
		log.Error("Failed to GetBlockBatches", "batch_id", id, "err", err)
		return
	}
	batch := batches[0]

	traces, err := r.db.GetBlockTraces(map[string]interface{}{"batch_id": id}, "ORDER BY number ASC")
	if err != nil || len(traces) == 0 {
		log.Error("Failed to GetBlockTraces", "batch_id", id, "err", err)
		return
	}

	layer2Batch := &bridge_abi.IZKRollupLayer2Batch{
		BatchIndex: batch.Index,
		ParentHash: common.HexToHash(batch.ParentHash),
		Blocks:     make([]bridge_abi.IZKRollupLayer2BlockHeader, len(traces)),
	}

	parentHash := common.HexToHash(batch.ParentHash)
	for i, trace := range traces {
		layer2Batch.Blocks[i] = bridge_abi.IZKRollupLayer2BlockHeader{
			BlockHash:   trace.Header.Hash(),
			ParentHash:  parentHash,
			BaseFee:     trace.Header.BaseFee,
			StateRoot:   trace.StorageTrace.RootAfter,
			BlockHeight: trace.Header.Number.Uint64(),
			GasUsed:     0,
			Timestamp:   trace.Header.Time,
			ExtraData:   make([]byte, 0),
			Txs:         make([]bridge_abi.IZKRollupLayer2Transaction, len(trace.Transactions)),
		}
		for j, tx := range trace.Transactions {
			layer2Batch.Blocks[i].Txs[j] = bridge_abi.IZKRollupLayer2Transaction{
				Caller:   tx.From,
				Nonce:    tx.Nonce,
				Gas:      tx.Gas,
				GasPrice: tx.GasPrice.ToInt(),
				Value:    tx.Value.ToInt(),
				Data:     common.Hex2Bytes(tx.Data),
				R:        tx.R.ToInt(),
				S:        tx.S.ToInt(),
				V:        tx.V.ToInt().Uint64(),
			}
			if tx.To != nil {
				layer2Batch.Blocks[i].Txs[j].Target = *tx.To
			}
			layer2Batch.Blocks[i].GasUsed += trace.ExecutionResults[j].Gas
		}

		// for next iteration
		parentHash = layer2Batch.Blocks[i].BlockHash
	}

	data, err := r.l1RollupABI.Pack("commitBatch", layer2Batch)
	if err != nil {
		log.Error("Failed to pack commitBatch", "id", id, "index", batch.Index, "err", err)
		return
	}

	txID := id + "-commit"
	// add suffix `-commit` to avoid duplication with finalize tx in unit tests
	hash, err := r.rollupSender.SendTransaction(txID, &r.cfg.RollupContractAddress, big.NewInt(0), data)
	if err != nil {
		if !errors.Is(err, sender.ErrNoAvailableAccount) {
			log.Error("Failed to send commitBatch tx to layer1 ", "id", id, "index", batch.Index, "err", err)
		}
		return
	}
	log.Info("commitBatch in layer1", "batch_id", id, "index", batch.Index, "hash", hash)

	// record and sync with db, @todo handle db error
	err = r.db.UpdateCommitTxHashAndRollupStatus(r.ctx, id, hash.String(), orm.RollupCommitting)
	if err != nil {
		log.Error("UpdateCommitTxHashAndRollupStatus failed", "id", id, "index", batch.Index, "err", err)
	}
	r.processingCommitment.Store(txID, id)
}

// ProcessCommittedBatches submit proof to layer 1 rollup contract
func (r *Layer2Relayer) ProcessCommittedBatches(wg *sync.WaitGroup) {
	defer wg.Done()

	// set skipped batches in a single db operation
	r.db.UpdateSkippedBatches()

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

		proofBuffer, instanceBuffer, err := r.db.GetVerifiedProofAndInstanceByID(id)
		if err != nil {
			log.Warn("fetch get proof by id failed", "id", id, "err", err)
			return
		}
		if proofBuffer == nil || instanceBuffer == nil {
			log.Warn("proof or instance not ready", "id", id)
			return
		}
		if len(proofBuffer)%32 != 0 {
			log.Error("proof buffer has wrong length", "id", id, "length", len(proofBuffer))
			return
		}
		if len(instanceBuffer)%32 != 0 {
			log.Warn("instance buffer has wrong length", "id", id, "length", len(instanceBuffer))
			return
		}

		proof := utils.BufferToUint256Le(proofBuffer)
		instance := utils.BufferToUint256Le(instanceBuffer)
		data, err := r.l1RollupABI.Pack("finalizeBatchWithProof", common.HexToHash(id), proof, instance)
		if err != nil {
			log.Error("Pack finalizeBatchWithProof failed", "err", err)
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

// Start the relayer process
func (r *Layer2Relayer) Start() {
	go func() {
		// trigger by timer
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				var wg = sync.WaitGroup{}
				wg.Add(3)
				go r.ProcessSavedEvents(&wg)
				go r.ProcessPendingBatches(&wg)
				go r.ProcessCommittedBatches(&wg)
				wg.Wait()
			case confirmation := <-r.messageCh:
				r.handleConfirmation(confirmation)
			case confirmation := <-r.rollupCh:
				r.handleConfirmation(confirmation)
			case <-r.stopCh:
				return
			}
		}
	}()
}

// Stop the relayer module, for a graceful shutdown.
func (r *Layer2Relayer) Stop() {
	close(r.stopCh)
}

func (r *Layer2Relayer) handleConfirmation(confirmation *sender.Confirmation) {
	if !confirmation.IsSuccessful {
		log.Warn("transaction confirmed but failed in layer1", "confirmation", confirmation)
		return
	}

	transactionType := "Unknown"
	// check whether it is message relay transaction
	if msgHash, ok := r.processingMessage.Load(confirmation.ID); ok {
		transactionType = "MessageRelay"
		// @todo handle db error
		err := r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, msgHash.(string), orm.MsgConfirmed, confirmation.TxHash.String())
		if err != nil {
			log.Warn("UpdateLayer2StatusAndLayer1Hash failed", "msgHash", msgHash.(string), "err", err)
		}
		r.processingMessage.Delete(confirmation.ID)
	}

	// check whether it is block commitment transaction
	if batchID, ok := r.processingCommitment.Load(confirmation.ID); ok {
		transactionType = "BatchCommitment"
		// @todo handle db error
		err := r.db.UpdateCommitTxHashAndRollupStatus(r.ctx, batchID.(string), confirmation.TxHash.String(), orm.RollupCommitted)
		if err != nil {
			log.Warn("UpdateCommitTxHashAndRollupStatus failed", "batch_id", batchID.(string), "err", err)
		}
		r.processingCommitment.Delete(confirmation.ID)
	}

	// check whether it is proof finalization transaction
	if batchID, ok := r.processingFinalization.Load(confirmation.ID); ok {
		transactionType = "ProofFinalization"
		// @todo handle db error
		err := r.db.UpdateFinalizeTxHashAndRollupStatus(r.ctx, batchID.(string), confirmation.TxHash.String(), orm.RollupFinalized)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "batch_id", batchID.(string), "err", err)
		}
		r.processingFinalization.Delete(confirmation.ID)
	}
	log.Info("transaction confirmed in layer1", "type", transactionType, "confirmation", confirmation)
}
