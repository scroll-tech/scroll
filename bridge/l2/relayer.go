package l2

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	// not sure if this will make problems when relay with l1geth

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
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
	sender *sender.Sender

	proofGenerationFreq uint64
	skippedOpcodes      map[string]struct{}

	db  database.OrmFactory
	cfg *config.RelayerConfig

	l1MessengerABI *abi.ABI
	l1RollupABI    *abi.ABI

	// a list of processing message, indexed by layer2 hash
	processingMessage map[string]string

	// a list of processing batch commitment, indexed by block height
	processingCommitment map[string]uint64

	// a list of processing batch finalization, indexed by block height
	processingFinalization map[string]uint64

	// channel used to communicate with transaction sender
	confirmationCh <-chan *sender.Confirmation
	stopCh         chan struct{}
}

// NewLayer2Relayer will return a new instance of Layer2RelayerClient
func NewLayer2Relayer(ctx context.Context, ethClient *ethclient.Client, proofGenFreq uint64, skippedOpcodes map[string]struct{}, l2ConfirmNum int64, db database.OrmFactory, cfg *config.RelayerConfig) (*Layer2Relayer, error) {

	l1MessengerABI, err := bridge_abi.L1MessengerMetaData.GetAbi()
	if err != nil {
		log.Error("Get L1MessengerABI failed", "err", err)
		return nil, err
	}

	l1RollupABI, err := bridge_abi.RollupMetaData.GetAbi()
	if err != nil {
		log.Error("Get RollupABI failed", "err", err)
		return nil, err
	}

	prv, err := crypto.HexToECDSA(cfg.PrivateKey)
	if err != nil {
		log.Error("Failed to import private key from config file")
		return nil, err
	}

	// @todo use different sender for relayer, block commit and proof finalize
	sender, err := sender.NewSender(ctx, cfg.SenderConfig, prv)
	if err != nil {
		log.Error("Failed to create sender", "err", err)
		return nil, err
	}

	return &Layer2Relayer{
		ctx:                    ctx,
		client:                 ethClient,
		sender:                 sender,
		db:                     db,
		l1MessengerABI:         l1MessengerABI,
		l1RollupABI:            l1RollupABI,
		cfg:                    cfg,
		proofGenerationFreq:    proofGenFreq,
		skippedOpcodes:         skippedOpcodes,
		processingMessage:      map[string]string{},
		processingCommitment:   map[string]uint64{},
		processingFinalization: map[string]uint64{},
		stopCh:                 make(chan struct{}),
		confirmationCh:         sender.ConfirmChan(),
	}, nil
}

// ProcessSavedEvents relays saved un-processed cross-domain transactions to desired blockchain
func (r *Layer2Relayer) ProcessSavedEvents() {
	// msgs are sorted by nonce in increasing order
	msgs, err := r.db.GetL2UnprocessedMessages()
	if err != nil {
		log.Error("Failed to fetch unprocessed L2 messages", "err", err)
		return
	}
	if len(msgs) == 0 {
		return
	}
	msg := msgs[0]
	// @todo add support to relay multiple messages
	batch_id, err := r.db.GetLatestFinalizedBatch()
	if err != nil {
		log.Error("GetLatestFinalizedBatch failed", "err", err)
		return
	}
	blocks, err := r.db.GetBlockInfos(map[string]interface{}{"batch_id": batch_id}, "ORDER BY number DESC")
	if err != nil || len(blocks) == 0 {
		log.Error("GetBlockResults failed", "batch_id", batch_id, "err", err)
		return
	}
	if blocks[0].Number < msg.Height {
		// log.Warn("corresponding block not finalized", "status", status)
		return
	}

	// @todo fetch merkle proof from l2geth
	log.Info("Processing L2 Message", "msg.nonce", msg.Nonce, "msg.height", msg.Height)

	proof := bridge_abi.IL1ScrollMessengerL2MessageProof{
		BlockNumber: big.NewInt(int64(msg.Height)),
		MerkleProof: make([]byte, 0),
	}
	sender := common.HexToAddress(msg.Sender)
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
	data, err := r.l1MessengerABI.Pack("relayMessageWithProof", sender, target, value, fee, deadline, msgNonce, calldata, proof)
	if err != nil {
		log.Error("Failed to pack relayMessageWithProof", "msg.nonce", msg.Nonce, "err", err)
		// TODO: need to skip this message by changing its status to MsgError
		return
	}

	hash, err := r.sender.SendTransaction(msg.Layer2Hash, &r.cfg.MessengerContractAddress, big.NewInt(0), data)
	if err != nil {
		log.Error("Failed to send relayMessageWithProof tx to L1", "err", err)
		return
	}
	log.Info("relayMessageWithProof to layer1", "layer2hash", msg.Layer2Hash, "txhash", hash.String())

	// save status in db
	// @todo handle db error
	err = r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, msg.Layer2Hash, hash.String(), orm.MsgSubmitted)
	if err != nil {
		log.Error("UpdateLayer2StatusAndLayer1Hash failed", "layer2hash", msg.Layer2Hash, "err", err)
	}
	r.processingMessage[msg.Layer2Hash] = msg.Layer2Hash
}

// ProcessPendingBatches submit batch data to layer 1 rollup contract
// TODO: this logic is definitely wrong
func (r *Layer2Relayer) ProcessPendingBatches() {
	// batches are sorted by id in increasing order
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

	// will fetch missing block result from l2geth
	trace, err := r.getOrFetchBlockResultByHeight(id)
	if err != nil {
		log.Error("getOrFetchBlockResultByHeight failed",
			"id", id,
			"err", err,
		)
		return
	}
	if trace == nil {
		return
	}

	// TODO: rethink about this?????
	parentHash, err := r.getOrFetchBlockHashByHeight(id - 1)
	if err != nil {
		log.Error("getOrFetchBlockHashByHeight for parent block failed",
			"parent id", id-1,
			"err", err,
		)
		return
	}
	if parentHash == nil {
		log.Error("parent hash is empty",
			"id", id,
			"err", err,
		)
		return
	}

	header := bridge_abi.IZKRollupBlockHeader{
		BlockHash:   trace.BlockTrace.Hash,
		ParentHash:  *parentHash,
		BaseFee:     trace.BlockTrace.BaseFee.ToInt(),
		StateRoot:   trace.StorageTrace.RootAfter,
		BlockHeight: trace.BlockTrace.Number.ToInt().Uint64(),
		GasUsed:     0,
		Timestamp:   trace.BlockTrace.Time,
		ExtraData:   make([]byte, 0),
	}
	txns := make([]bridge_abi.IZKRollupLayer2Transaction, len(trace.BlockTrace.Transactions))
	for i, tx := range trace.BlockTrace.Transactions {
		txns[i] = bridge_abi.IZKRollupLayer2Transaction{
			Caller:   tx.From,
			Nonce:    tx.Nonce,
			Gas:      tx.Gas,
			GasPrice: tx.GasPrice.ToInt(),
			Value:    tx.Value.ToInt(),
			Data:     common.Hex2Bytes(tx.Data),
		}
		if tx.To != nil {
			txns[i].Target = *tx.To
		}
		header.GasUsed += trace.ExecutionResults[i].Gas
	}

	data, err := r.l1RollupABI.Pack("commitBlock", header, txns)
	if err != nil {
		log.Error("Failed to pack commitBatch", "id", id, "err", err)
		return
	}
	hash, err := r.sender.SendTransaction(strconv.FormatUint(id, 10), &r.cfg.RollupContractAddress, big.NewInt(0), data)
	if err != nil {
		log.Error("Failed to send commitBatch tx to layer1 ", "id", id, "err", err)
		return
	}
	log.Info("commitBatch in layer1", "id", id, "hash", hash)

	// record and sync with db, @todo handle db error
	err = r.db.UpdateRollupTxHashAndRollupStatus(r.ctx, id, hash.String(), orm.RollupCommitting)
	if err != nil {
		log.Error("UpdateRollupTxHashAndRollupStatus failed", "id", id, "err", err)
	}
	r.processingCommitment[strconv.FormatUint(id, 10)] = id
}

// ProcessCommittedBatches submit proof to layer rollup contract
func (r *Layer2Relayer) ProcessCommittedBatches() {
	// batches are sorted by id in increasing order
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

		proof := bufferToUint256Le(proofBuffer)
		instance := bufferToUint256Le(instanceBuffer)

		// it must in db
		hash, err := r.db.GetHashByNumber(id)
		if err != nil {
			log.Warn("fetch missing block result by id failed", "id", id, "err", err)
		}
		if hash == nil {
			// only happen when trace validate failed
			return
		}
		data, err := r.l1RollupABI.Pack("finalizeBlockWithProof", hash, proof, instance)
		if err != nil {
			log.Error("Pack finalizeBlockWithProof failed", err)
			return
		}
		txHash, err := r.sender.SendTransaction(strconv.FormatUint(id, 10), &r.cfg.RollupContractAddress, big.NewInt(0), data)
		hash = &txHash
		if err != nil {
			log.Error("finalizeBlockWithProof in layer1 failed",
				"id", id,
				"err", err,
			)
			return
		}
		log.Info("finalizeBlockWithProof in layer1", "id", id, "hash", hash)

		// record and sync with db, @todo handle db error
		err = r.db.UpdateFinalizeTxHashAndRollupStatus(r.ctx, id, hash.String(), orm.RollupFinalizing)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "id", id, "err", err)
		}
		success = true
		r.processingFinalization[strconv.FormatUint(id, 10)] = id

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
				r.ProcessSavedEvents()
				r.ProcessPendingBatches()
				r.ProcessCommittedBatches()
			case confirmation := <-r.confirmationCh:
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

func (r *Layer2Relayer) getOrFetchBlockHashByHeight(height uint64) (*common.Hash, error) {
	hash, err := r.db.GetHashByNumber(height)
	if err != nil {
		block, err := r.client.BlockByNumber(r.ctx, big.NewInt(int64(height)))
		if err != nil {
			return nil, err
		}
		x := block.Hash()
		return &x, err
	}
	return hash, nil
}

func (r *Layer2Relayer) getOrFetchBlockResultByHeight(height uint64) (*types.BlockResult, error) {
	tracesInDB, err := r.db.GetBlockResults(map[string]interface{}{"number": height})
	if err != nil {
		log.Warn("GetBlockResults failed", "height", height, "err", err)
		return nil, err
	}
	if len(tracesInDB) == 0 {
		return r.fetchMissingBlockResultByHeight(height)
	}
	return tracesInDB[0], nil
}

func (r *Layer2Relayer) fetchMissingBlockResultByHeight(height uint64) (*types.BlockResult, error) {
	header, err := r.client.HeaderByNumber(r.ctx, big.NewInt(int64(height)))
	if err != nil {
		return nil, err
	}
	trace, err := r.client.GetBlockResultByHash(r.ctx, header.Hash())
	if err != nil {
		return nil, err
	}
	if blockTraceIsValid(trace) {
		if err = r.db.InsertBlockResults(r.ctx, []*types.BlockResult{trace}); err != nil {
			log.Error("failed to store missing blockResult", "height", height, "err", err)
		}
		return trace, nil
	}
	return nil, fmt.Errorf("fetched block_trace is invalid, height: %d", height)
}

func (r *Layer2Relayer) handleConfirmation(confirmation *sender.Confirmation) {
	transactionType := "Unknown"
	// check whether it is message relay transaction
	if layer2Hash, ok := r.processingMessage[confirmation.ID]; ok {
		transactionType = "MessageRelay"
		// @todo handle db error
		err := r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, layer2Hash, confirmation.TxHash.String(), orm.MsgConfirmed)
		if err != nil {
			log.Warn("UpdateLayer2StatusAndLayer1Hash failed", "layer2Hash", layer2Hash, "err", err)
		}
		delete(r.processingMessage, confirmation.ID)
	}

	// check whether it is block commitment transaction
	if batch_id, ok := r.processingCommitment[confirmation.ID]; ok {
		transactionType = "BlockCommitment"
		// @todo handle db error
		err := r.db.UpdateRollupTxHashAndRollupStatus(r.ctx, batch_id, confirmation.TxHash.String(), orm.RollupCommitted)
		if err != nil {
			log.Warn("UpdateRollupTxHashAndRollupStatus failed", "batch_id", batch_id, "err", err)
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

		// try to delete block trace
		err = r.db.DeleteTracesByBatchID(batch_id)
		if err != nil {
			log.Warn("DeleteTracesByBatchID failed", "batch_id", batch_id, "err", err)
		}
	}
	log.Info("transaction confirmed in layer1", "type", transactionType, "confirmation", confirmation)
}

//nolint:unused
func bufferToUint256Be(buffer []byte) []*big.Int {
	buffer256 := make([]*big.Int, len(buffer)/32)
	for i := 0; i < len(buffer)/32; i++ {
		buffer256[i] = big.NewInt(0)
		for j := 0; j < 32; j++ {
			buffer256[i] = buffer256[i].Lsh(buffer256[i], 8)
			buffer256[i] = buffer256[i].Add(buffer256[i], big.NewInt(int64(buffer[i*32+j])))
		}
	}
	return buffer256
}

func bufferToUint256Le(buffer []byte) []*big.Int {
	buffer256 := make([]*big.Int, len(buffer)/32)
	for i := 0; i < len(buffer)/32; i++ {
		v := big.NewInt(0)
		shft := big.NewInt(1)
		for j := 0; j < 32; j++ {
			v = new(big.Int).Add(v, new(big.Int).Mul(shft, big.NewInt(int64(buffer[i*32+j]))))
			shft = new(big.Int).Mul(shft, big.NewInt(256))
		}
		buffer256[i] = v
	}
	return buffer256
}
