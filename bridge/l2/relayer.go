package l2

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strconv"
	"time"

	// not sure if this will make problems when relay with l1geth

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
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

	proofGenerationFreq uint64
	skippedOpcodes      map[string]struct{}

	db  database.OrmFactory
	cfg *config.RelayerConfig

	messengerSender *sender.Sender
	messengerCh     <-chan *sender.Confirmation
	l1MessengerABI  *abi.ABI

	rollupSender *sender.Sender
	rollupCh     <-chan *sender.Confirmation
	l1RollupABI  *abi.ABI

	// a list of processing message, indexed by layer2 hash
	processingMessage map[string]string

	// a list of processing block, indexed by block height
	processingBlock map[string]uint64

	// a list of processing proof, indexed by block height
	processingProof map[string]uint64

	stopCh chan struct{}
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

	// @todo use different sender for relayer, block commit and proof finalize
	messengerSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.PrivateKeyList)
	if err != nil {
		log.Error("Failed to create messenger sender", "err", err)
		return nil, err
	}

	rollupSender, err := sender.NewSender(ctx, cfg.SenderConfig, []*ecdsa.PrivateKey{cfg.RollerPrivateKey})
	if err != nil {
		log.Error("Failed to create rollup sender", "err", err)
		return nil, err
	}

	return &Layer2Relayer{
		ctx:                 ctx,
		client:              ethClient,
		db:                  db,
		messengerSender:     messengerSender,
		messengerCh:         messengerSender.ConfirmChan(),
		l1MessengerABI:      l1MessengerABI,
		rollupSender:        rollupSender,
		rollupCh:            rollupSender.ConfirmChan(),
		l1RollupABI:         l1RollupABI,
		cfg:                 cfg,
		proofGenerationFreq: proofGenFreq,
		skippedOpcodes:      skippedOpcodes,
		processingMessage:   map[string]string{},
		processingBlock:     map[string]uint64{},
		processingProof:     map[string]uint64{},
		stopCh:              make(chan struct{}),
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
	for _, msg := range msgs {
		if err := r.processSavedEvent(msg); err != nil {
			log.Error("failed to process l2 saved event", "err", err)
			return
		}
	}
}

func (r *Layer2Relayer) processSavedEvent(msg *orm.Layer2Message) error {
	// @todo add support to relay multiple messages
	latestFinalizeHeight, err := r.db.GetLatestFinalizedBlock()
	if err != nil {
		log.Error("GetLatestFinalizedBlock failed", "err", err)
		return err
	}
	if latestFinalizeHeight < msg.Height {
		// log.Warn("corresponding block not finalized", "status", status)
		return nil
	}

	// @todo fetch merkle proof from l2geth
	log.Info("Processing L2 Message", "msg.nonce", msg.Nonce, "msg.height", msg.Height)

	proof := bridge_abi.IL1ScrollMessengerL2MessageProof{
		BlockNumber: big.NewInt(int64(msg.Height)),
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

	hash, err := r.messengerSender.SendTransaction(msg.Layer2Hash, &r.cfg.MessengerContractAddress, big.NewInt(0), data)
	if err != nil {
		log.Error("Failed to send relayMessageWithProof tx to L1", "err", err)
		return err
	}
	log.Info("relayMessageWithProof to layer1", "layer2hash", msg.Layer2Hash, "txhash", hash.String())

	// save status in db
	// @todo handle db error
	err = r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, msg.Layer2Hash, hash.String(), orm.MsgSubmitted)
	if err != nil {
		log.Error("UpdateLayer2StatusAndLayer1Hash failed", "layer2hash", msg.Layer2Hash, "err", err)
	}
	r.processingMessage[msg.Layer2Hash] = msg.Layer2Hash
	return err
}

// ProcessPendingBlocks submit block data to layer rollup contract
func (r *Layer2Relayer) ProcessPendingBlocks() {
	// blocks are sorted by height in increasing order
	blocksInDB, err := r.db.GetPendingBlocks()
	if err != nil {
		log.Error("Failed to fetch pending L2 blocks", "err", err)
		return
	}
	if len(blocksInDB) == 0 {
		return
	}
	height := blocksInDB[0]
	// @todo add support to relay multiple blocks

	// will fetch missing block result from l2geth
	trace, err := r.getOrFetchBlockResultByHeight(height)
	if err != nil {
		log.Error("getOrFetchBlockResultByHeight failed",
			"height", height,
			"err", err,
		)
		return
	}
	if trace == nil {
		return
	}
	parentHash, err := r.getOrFetchBlockHashByHeight(height - 1)
	if err != nil {
		log.Error("getOrFetchBlockHashByHeight for parent block failed",
			"parent height", height-1,
			"err", err,
		)
		return
	}
	if parentHash == nil {
		log.Error("parent hash is empty",
			"height", height,
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
		log.Error("Failed to pack commitBlock", "height", height, "err", err)
		return
	}
	hash, err := r.rollupSender.SendTransaction(strconv.FormatUint(height, 10), &r.cfg.RollupContractAddress, big.NewInt(0), data)
	if err != nil {
		log.Error("Failed to send commitBlock tx to layer1 ", "height", height, "err", err)
		return
	}
	log.Info("commitBlock in layer1", "height", height, "hash", hash)

	// record and sync with db, @todo handle db error
	err = r.db.UpdateRollupTxHashAndStatus(r.ctx, height, hash.String(), orm.RollupCommitting)
	if err != nil {
		log.Error("UpdateRollupTxHashAndStatus failed", "height", height, "err", err)
	}
	r.processingBlock[strconv.FormatUint(height, 10)] = height
}

// ProcessCommittedBlocks submit proof to layer rollup contract
func (r *Layer2Relayer) ProcessCommittedBlocks() {
	// blocks are sorted by height in increasing order
	blocksInDB, err := r.db.GetCommittedBlocks()
	if err != nil {
		log.Error("Failed to fetch committed L2 blocks", "err", err)
		return
	}
	if len(blocksInDB) == 0 {
		return
	}
	height := blocksInDB[0]
	// @todo add support to relay multiple blocks

	status, err := r.db.GetBlockStatusByNumber(height)
	if err != nil {
		log.Error("GetBlockStatusByNumber failed", "height", height, "err", err)
		return
	}

	switch status {
	case orm.BlockUnassigned, orm.BlockAssigned:
		// The proof for this block is not ready yet.
		return

	case orm.BlockProved:
		// It's an intermediate state. The roller manager received the proof but has not verified
		// the proof yet. We don't roll up the proof until it's verified.
		return

	case orm.BlockFailed, orm.BlockSkipped:
		if err = r.db.UpdateRollupStatus(r.ctx, height, orm.RollupFinalizationSkipped); err != nil {
			log.Warn("UpdateRollupStatus failed", "height", height, "err", err)
		}

	case orm.BlockVerified:
		log.Info("Start to roll up zk proof", "height", height)
		success := false

		defer func() {
			// TODO: need to revisit this and have a more fine-grained error handling
			if !success {
				log.Info("Failed to upload the proof, change rollup status to FinalizationSkipped", "height", height)
				if err = r.db.UpdateRollupStatus(r.ctx, height, orm.RollupFinalizationSkipped); err != nil {
					log.Warn("UpdateRollupStatus failed", "height", height, "err", err)
				}
			}
		}()

		proofBuffer, instanceBuffer, err := r.db.GetVerifiedProofAndInstanceByNumber(height)
		if err != nil {
			log.Warn("fetch get proof by height failed", "height", height, "err", err)
			return
		}
		if proofBuffer == nil || instanceBuffer == nil {
			log.Warn("proof or instance not ready", "height", height)
			return
		}
		if len(proofBuffer)%32 != 0 {
			log.Error("proof buffer has wrong length", "height", height, "length", len(proofBuffer))
			return
		}
		if len(instanceBuffer)%32 != 0 {
			log.Warn("instance buffer has wrong length", "height", height, "length", len(instanceBuffer))
			return
		}

		proof := bufferToUint256Le(proofBuffer)
		instance := bufferToUint256Le(instanceBuffer)

		// it must in db
		hash, err := r.db.GetHashByNumber(height)
		if err != nil {
			log.Warn("fetch missing block result by height failed", "height", height, "err", err)
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
		txHash, err := r.rollupSender.SendTransaction(strconv.FormatUint(height, 10), &r.cfg.RollupContractAddress, big.NewInt(0), data)
		hash = &txHash
		if err != nil {
			log.Error("finalizeBlockWithProof in layer1 failed",
				"height", height,
				"err", err,
			)
			return
		}
		log.Info("finalizeBlockWithProof in layer1", "height", height, "hash", hash)

		// record and sync with db, @todo handle db error
		err = r.db.UpdateFinalizeTxHashAndStatus(r.ctx, height, hash.String(), orm.RollupFinalizing)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndStatus failed", "height", height, "err", err)
		}
		success = true
		r.processingProof[strconv.FormatUint(height, 10)] = height

	default:
		log.Error("encounter unreachable case in ProcessCommittedBlocks",
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
				r.ProcessPendingBlocks()
				r.ProcessCommittedBlocks()
			case confirmation := <-r.messengerCh:
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
		// skip verify for unsupported block
		skip := false
		if height%r.proofGenerationFreq != 0 {
			log.Info("skip proof generation", "block", height)
			skip = true
		} else if TraceHasUnsupportedOpcodes(r.skippedOpcodes, trace) {
			log.Info("block has unsupported opcodes, skip proof generation", "block", height)
			skip = true
		}
		if skip {
			if err = r.db.InsertBlockResultsWithStatus(r.ctx, []*types.BlockResult{trace}, orm.BlockSkipped); err != nil {
				log.Error("failed to store missing blockResult", "err", err)
			}
		} else {
			if err = r.db.InsertBlockResultsWithStatus(r.ctx, []*types.BlockResult{trace}, orm.BlockUnassigned); err != nil {
				log.Error("failed to store missing blockResult", "err", err)
			}
		}
		return trace, nil
	}
	return nil, nil
}

func (r *Layer2Relayer) handleConfirmation(confirmation *sender.Confirmation) {
	transactionType := "Unknown"
	// check whether it is message relay transaction
	if layer2Hash, ok := r.processingMessage[confirmation.ID]; ok {
		transactionType = "MessageRelay"
		// @todo handle db error
		err := r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, layer2Hash, confirmation.TxHash.String(), orm.MsgConfirmed)
		if err != nil {
			log.Warn("UpdateLayer2StatusAndLayer1Hash failed", "err", err)
		}
		delete(r.processingMessage, confirmation.ID)
	}

	// check whether it is block commitment transaction
	if blockHeight, ok := r.processingBlock[confirmation.ID]; ok {
		transactionType = "BlockCommitment"
		// @todo handle db error
		err := r.db.UpdateRollupTxHashAndStatus(r.ctx, blockHeight, confirmation.TxHash.String(), orm.RollupCommitted)
		if err != nil {
			log.Warn("UpdateRollupTxHashAndStatus failed", "err", err)
		}
		delete(r.processingBlock, confirmation.ID)
	}

	// check whether it is proof finalization transaction
	if blockHeight, ok := r.processingProof[confirmation.ID]; ok {
		transactionType = "ProofFinalization"
		// @todo handle db error
		err := r.db.UpdateFinalizeTxHashAndStatus(r.ctx, blockHeight, confirmation.TxHash.String(), orm.RollupFinalized)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndStatus failed", "err", err)
		}
		delete(r.processingProof, confirmation.ID)

		// try to delete block trace
		err = r.db.DeleteTraceByNumber(blockHeight)
		if err != nil {
			log.Warn("DeleteTraceByNumber failed", "err", err)
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
