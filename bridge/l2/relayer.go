package l2

import (
	"context"
	"errors"
	"math/big"
	"time"

	// not sure if this will make problems when relay with l1geth

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/common/viper"

	bridge_abi "scroll-tech/bridge/abi"
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
	ctx    context.Context
	client *ethclient.Client

	db database.OrmFactory
	vp *viper.Viper

	messageSender  *sender.Sender
	messageCh      <-chan *sender.Confirmation
	l1MessengerABI *abi.ABI

	rollupSender *sender.Sender
	rollupCh     <-chan *sender.Confirmation
	l1RollupABI  *abi.ABI

	// a list of processing message, indexed by layer2 hash
	processingMessage map[string]string

	// a list of processing batch commitment, indexed by batch id
	processingCommitment map[string]string

	// a list of processing batch finalization, indexed by batch id
	processingFinalization map[string]string

	stopCh chan struct{}
}

// NewLayer2Relayer will return a new instance of Layer2RelayerClient
func NewLayer2Relayer(ctx context.Context, ethClient *ethclient.Client, db database.OrmFactory, vp *viper.Viper) (*Layer2Relayer, error) {
	// @todo use different sender for relayer, block commit and proof finalize
	messageSenderPrivateKeys := vp.GetECDSAKeys("message_sender_private_keys")
	rollupSenderPrivateKeys := vp.GetECDSAKeys("rollup_sender_private_keys")
	messageSender, err := sender.NewSender(ctx, vp.Sub("sender_config"), messageSenderPrivateKeys)
	if err != nil {
		log.Error("Failed to create messenger sender", "err", err)
		return nil, err
	}

	rollupSender, err := sender.NewSender(ctx, vp.Sub("sender_config"), rollupSenderPrivateKeys)
	if err != nil {
		log.Error("Failed to create rollup sender", "err", err)
		return nil, err
	}

	return &Layer2Relayer{
		ctx:                    ctx,
		client:                 ethClient,
		db:                     db,
		messageSender:          messageSender,
		messageCh:              messageSender.ConfirmChan(),
		l1MessengerABI:         bridge_abi.L1MessengerMetaABI,
		rollupSender:           rollupSender,
		rollupCh:               rollupSender.ConfirmChan(),
		l1RollupABI:            bridge_abi.RollupMetaABI,
		vp:                     vp,
		processingMessage:      map[string]string{},
		processingCommitment:   map[string]string{},
		processingFinalization: map[string]string{},
		stopCh:                 make(chan struct{}),
	}, nil
}

// ProcessSavedEvents relays saved un-processed cross-domain transactions to desired blockchain
func (r *Layer2Relayer) ProcessSavedEvents() {
	// msgs are sorted by nonce in increasing order
	msgs, err := r.db.GetL2MessagesByStatus(orm.MsgPending)
	if err != nil {
		log.Error("Failed to fetch unprocessed L2 messages", "err", err)
		return
	}
	for _, msg := range msgs {
		if err := r.processSavedEvent(msg); err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) {
				log.Error("failed to process l2 saved event", "err", err)
			}
			return
		}
	}
}

func (r *Layer2Relayer) processSavedEvent(msg *orm.L2Message) error {
	// @todo add support to relay multiple messages
	batch, err := r.db.GetLatestFinalizedBatch()
	if err != nil {
		log.Error("GetLatestFinalizedBatch failed", "err", err)
		return err
	}

	if batch.EndBlockNumber < msg.Height {
		// log.Warn("corresponding block not finalized", "status", status)
		return nil
	}

	// @todo fetch merkle proof from l2geth
	log.Info("Processing L2 Message", "msg.nonce", msg.Nonce, "msg.height", msg.Height)

	proof := bridge_abi.IL1ScrollMessengerL2MessageProof{
		BlockHeight: big.NewInt(int64(msg.Height)),
		BatchIndex:  big.NewInt(int64(batch.Index)),
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

	messengerContractAddress := r.vp.GetAddress("messenger_contract_address")
	hash, err := r.messageSender.SendTransaction(msg.MsgHash, &messengerContractAddress, big.NewInt(0), data)
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
	r.processingMessage[msg.MsgHash] = msg.MsgHash
	return nil
}

// ProcessPendingBatches submit batch data to layer 1 rollup contract
func (r *Layer2Relayer) ProcessPendingBatches() {
	// batches are sorted by batch index in increasing order
	batchesInDB, err := r.db.GetPendingBatches()
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

	rollupContractAddress := r.vp.GetAddress("rollup_contract_address")
	hash, err := r.rollupSender.SendTransaction(id, &rollupContractAddress, big.NewInt(0), data)
	if err != nil {
		if !errors.Is(err, sender.ErrNoAvailableAccount) {
			log.Error("Failed to send commitBatch tx to layer1 ", "id", id, "index", batch.Index, "err", err)
		}
		return
	}
	log.Info("commitBatch in layer1", "id", id, "index", batch.Index, "hash", hash)

	// record and sync with db, @todo handle db error
	err = r.db.UpdateCommitTxHashAndRollupStatus(r.ctx, id, hash.String(), orm.RollupCommitting)
	if err != nil {
		log.Error("UpdateCommitTxHashAndRollupStatus failed", "id", id, "index", batch.Index, "err", err)
	}
	r.processingCommitment[id] = id
}

// ProcessCommittedBatches submit proof to layer 1 rollup contract
func (r *Layer2Relayer) ProcessCommittedBatches() {
	// batches are sorted by batch index in increasing order
	batches, err := r.db.GetCommittedBatches()
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

		rollupContractAddress := r.vp.GetAddress("rollup_contract_address")
		txHash, err := r.rollupSender.SendTransaction(id, &rollupContractAddress, big.NewInt(0), data)
		hash := &txHash
		if err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) {
				log.Error("finalizeBatchWithProof in layer1 failed", "id", id, "err", err)
			}
			return
		}
		log.Info("finalizeBatchWithProof in layer1", "id", id, "hash", hash)

		// record and sync with db, @todo handle db error
		err = r.db.UpdateFinalizeTxHashAndRollupStatus(r.ctx, id, hash.String(), orm.RollupFinalizing)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "id", id, "err", err)
		}
		success = true
		r.processingFinalization[id] = id

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
		relayerLoopTimeSec := r.vp.GetInt("relayer_loop_time_sec")
		ticker := time.NewTicker(time.Duration(relayerLoopTimeSec) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.ProcessSavedEvents()
				r.ProcessPendingBatches()
				r.ProcessCommittedBatches()
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
	if msgHash, ok := r.processingMessage[confirmation.ID]; ok {
		transactionType = "MessageRelay"
		// @todo handle db error
		err := r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, msgHash, orm.MsgConfirmed, confirmation.TxHash.String())
		if err != nil {
			log.Warn("UpdateLayer2StatusAndLayer1Hash failed", "msgHash", msgHash, "err", err)
		}
		delete(r.processingMessage, confirmation.ID)
	}

	// check whether it is block commitment transaction
	if batch_id, ok := r.processingCommitment[confirmation.ID]; ok {
		transactionType = "BatchCommitment"
		// @todo handle db error
		err := r.db.UpdateCommitTxHashAndRollupStatus(r.ctx, batch_id, confirmation.TxHash.String(), orm.RollupCommitted)
		if err != nil {
			log.Warn("UpdateCommitTxHashAndRollupStatus failed", "batch_id", batch_id, "err", err)
		}
		delete(r.processingCommitment, confirmation.ID)
	}

	// check whether it is proof finalization transaction
	if batch_id, ok := r.processingFinalization[confirmation.ID]; ok {
		transactionType = "ProofFinalization"
		// @todo handle db error
		err := r.db.UpdateFinalizeTxHashAndRollupStatus(r.ctx, batch_id, confirmation.TxHash.String(), orm.RollupFinalized)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "batch_id", batch_id, "err", err)
		}
		delete(r.processingFinalization, confirmation.ID)
	}
	log.Info("transaction confirmed in layer1", "type", transactionType, "confirmation", confirmation)
}
