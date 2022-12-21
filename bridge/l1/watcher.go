package l1

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/utils"
)

type relayedMessage struct {
	msgHash      common.Hash
	txHash       common.Hash
	isSuccessful bool
}

type rollupEvent struct {
	batchID common.Hash
	txHash  common.Hash
	status  orm.RollupStatus
}

// Watcher will listen for smart contract events from Eth L1.
type Watcher struct {
	ctx    context.Context
	client *ethclient.Client
	db     database.OrmFactory

	// The number of new blocks to wait for a block to be confirmed
	confirmations uint64

	messengerAddress common.Address
	messengerABI     *abi.ABI

	messageQueueAddress common.Address
	messageQueueABI     *abi.ABI

	rollupAddress common.Address
	rollupABI     *abi.ABI

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64
	// The height of the block that the watcher has retrieved header rlp
	processedBlockHeight uint64

	stop chan bool
}

// NewWatcher returns a new instance of Watcher. The instance will be not fully prepared,
// and still needs to be finalized and ran by calling `watcher.Start`.
func NewWatcher(ctx context.Context, client *ethclient.Client, startHeight uint64, confirmations uint64, messengerAddress common.Address, messageQueueAddress common.Address, rollupAddress common.Address, db database.OrmFactory) (*Watcher, error) {
	savedMsgHeight, err := db.GetLayer1LatestWatchedHeight()
	if err != nil {
		log.Warn("Failed to fetch L1 watched message height from db", "err", err)
		return nil, err
	}
	if savedMsgHeight < int64(startHeight) {
		savedMsgHeight = int64(startHeight)
	}
	savedBlockHeight, err := db.GetLatestL1BlockHeight()
	if err != nil {
		log.Warn("Failed to fetch latest L1 block height from db", "err", err)
		return nil, err
	}
	if savedBlockHeight < startHeight {
		savedBlockHeight = startHeight
	}

	stop := make(chan bool)

	return &Watcher{
		ctx:           ctx,
		client:        client,
		db:            db,
		confirmations: confirmations,

		messengerAddress: messengerAddress,
		messengerABI:     bridge_abi.L1MessengerABI,

		messageQueueAddress: messageQueueAddress,
		messageQueueABI:     bridge_abi.L1MessageQueueABI,

		rollupAddress: rollupAddress,
		rollupABI:     bridge_abi.RollupABI,

		processedMsgHeight:   uint64(savedMsgHeight),
		processedBlockHeight: savedBlockHeight,

		stop: stop,
	}, nil
}

// Start the Watcher module.
func (w *Watcher) Start() {
	go func() {
		// trigger by timer
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				blockNumber, err := w.client.BlockNumber(w.ctx)
				if err != nil {
					log.Error("Failed to get block number", "err", err)
				}

				if err := w.fetchBlockHeader(blockNumber); err != nil {
					log.Error("Failed to fetch L1 block header", "lastest", blockNumber, "err", err)
				}

				if err := w.fetchContractEvent(blockNumber); err != nil {
					log.Error("Failed to fetch L1 bridge event", "lastest", blockNumber, "err", err)
				}
			case <-w.stop:
				return
			}
		}
	}()
}

// Stop the Watcher module, for a graceful shutdown.
func (w *Watcher) Stop() {
	w.stop <- true
}

const contractEventsBlocksFetchLimit = int64(10)

// fetchBlockHeader pull latest L1 blocks and save in DB
func (w *Watcher) fetchBlockHeader(blockHeight uint64) error {
	fromBlock := int64(w.processedBlockHeight) + 1
	toBlock := int64(blockHeight) - int64(w.confirmations)
	if toBlock < fromBlock {
		return nil
	}
	if toBlock > fromBlock+contractEventsBlocksFetchLimit {
		toBlock = fromBlock + contractEventsBlocksFetchLimit - 1
	}

	var blocks []*orm.L1BlockInfo
	var err error
	height := fromBlock
	for ; height <= toBlock; height++ {
		var block *types.Block
		block, err = w.client.BlockByNumber(w.ctx, big.NewInt(height))
		if err != nil {
			log.Warn("Failed to get block", "height", height, "err", err)
			break
		}
		var headerRLPBytes []byte
		headerRLPBytes, err = rlp.EncodeToBytes(block.Header())
		if err != nil {
			log.Warn("Failed to rlp encode header", "height", height, "err", err)
			break
		}
		blocks = append(blocks, &orm.L1BlockInfo{
			Number:    uint64(height),
			Hash:      block.Hash().String(),
			HeaderRLP: common.Bytes2Hex(headerRLPBytes),
		})
	}

	// failed at first block, return with the error
	if height == fromBlock {
		return err
	}
	toBlock = height - 1

	// insert succeed blocks
	err = w.db.InsertL1Blocks(w.ctx, blocks)
	if err != nil {
		log.Warn("Failed to insert L1 block to db", "fromBlock", fromBlock, "toBlock", toBlock, "err", err)
		return err
	}

	// update processed height
	w.processedBlockHeight = uint64(toBlock)
	return nil
}

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (w *Watcher) fetchContractEvent(blockHeight uint64) error {
	fromBlock := int64(w.processedMsgHeight) + 1
	toBlock := int64(blockHeight) - int64(w.confirmations)

	if toBlock < fromBlock {
		return nil
	}

	if toBlock > fromBlock+contractEventsBlocksFetchLimit {
		toBlock = fromBlock + contractEventsBlocksFetchLimit - 1
	}

	// warning: uint int conversion...
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlock), // inclusive
		ToBlock:   big.NewInt(toBlock),   // inclusive
		Addresses: []common.Address{
			w.messengerAddress,
			w.messageQueueAddress,
			w.rollupAddress,
		},
		Topics: make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 6)
	query.Topics[0][0] = bridge_abi.L1SendMessageEventSignature
	query.Topics[0][1] = bridge_abi.L1RelayedMessageEventSignature
	query.Topics[0][2] = bridge_abi.L1FailedRelayedMessageEventSignature
	query.Topics[0][3] = bridge_abi.L1CommitBatchEventSignature
	query.Topics[0][4] = bridge_abi.L1FinalizeBatchEventSignature
	query.Topics[0][5] = bridge_abi.L1AppendMessageEventSignature

	logs, err := w.client.FilterLogs(w.ctx, query)
	if err != nil {
		log.Warn("Failed to get event logs", "err", err)
		return err
	}
	if len(logs) == 0 {
		w.processedMsgHeight = uint64(toBlock)
		return nil
	}
	log.Info("Received new L1 messages", "fromBlock", fromBlock, "toBlock", toBlock,
		"cnt", len(logs))

	sentMessageEvents, relayedMessageEvents, rollupEvents, err := w.parseBridgeEventLogs(logs)
	if err != nil {
		log.Error("Failed to parse emitted events log", "err", err)
		return err
	}

	// use rollup event to update rollup results db status
	var batchIDs []string
	for _, event := range rollupEvents {
		batchIDs = append(batchIDs, event.batchID.String())
	}
	statuses, err := w.db.GetRollupStatusByIDList(batchIDs)
	if err != nil {
		log.Error("Failed to GetRollupStatusByIDList", "err", err)
		return err
	}
	if len(statuses) != len(batchIDs) {
		log.Error("RollupStatus.Length mismatch with BatchIDs.Length")
		return nil
	}

	for index, event := range rollupEvents {
		batchID := event.batchID.String()
		status := statuses[index]
		if event.status != status {
			if event.status == orm.RollupFinalized {
				err = w.db.UpdateFinalizeTxHashAndRollupStatus(w.ctx, batchID, event.txHash.String(), event.status)
			} else if event.status == orm.RollupCommitted {
				err = w.db.UpdateCommitTxHashAndRollupStatus(w.ctx, batchID, event.txHash.String(), event.status)
			}
			if err != nil {
				log.Error("Failed to update Rollup/Finalize TxHash and Status", "err", err)
				return err
			}
		}
	}

	// Update relayed message first to make sure we don't forget to update submitted message.
	// Since, we always start sync from the latest unprocessed message.
	for _, msg := range relayedMessageEvents {
		if msg.isSuccessful {
			// succeed
			err = w.db.UpdateLayer1StatusAndLayer2Hash(w.ctx, msg.msgHash.String(), orm.MsgConfirmed, msg.txHash.String())
		} else {
			// failed
			err = w.db.UpdateLayer1StatusAndLayer2Hash(w.ctx, msg.msgHash.String(), orm.MsgFailed, msg.txHash.String())
		}
		if err != nil {
			log.Error("Failed to update layer1 status and layer2 hash", "err", err)
			return err
		}
	}

	err = w.db.SaveL1Messages(w.ctx, sentMessageEvents)
	if err == nil {
		w.processedMsgHeight = uint64(toBlock)
	}
	return err
}

func (w *Watcher) parseBridgeEventLogs(logs []types.Log) ([]*orm.L1Message, []relayedMessage, []rollupEvent, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l1Messages []*orm.L1Message
	var relayedMessages []relayedMessage
	var rollupEvents []rollupEvent
	var lastAppendMsgHash common.Hash
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case bridge_abi.L1SendMessageEventSignature:
			event := struct {
				Target       common.Address
				Sender       common.Address
				Value        *big.Int // uint256
				Fee          *big.Int // uint256
				Deadline     *big.Int // uint256
				Message      []byte
				MessageNonce *big.Int // uint256
				GasLimit     *big.Int // uint256
			}{}
			err := utils.UnpackLog(w.messengerABI, event, "SentMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer1 SentMessage event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}
			computedMsgHash := utils.ComputeMessageHash(
				event.Target,
				event.Sender,
				event.Value,
				event.Fee,
				event.Deadline,
				event.Message,
				event.MessageNonce,
			)
			// they should always match, just double check
			if computedMsgHash != lastAppendMsgHash {
				return l1Messages, relayedMessages, rollupEvents, errors.New("l1 message hash mismatch")
			}

			l1Messages = append(l1Messages, &orm.L1Message{
				Nonce:      event.MessageNonce.Uint64(),
				MsgHash:    computedMsgHash.String(),
				Height:     vLog.BlockNumber,
				Sender:     event.Sender.String(),
				Value:      event.Value.String(),
				Fee:        event.Fee.String(),
				GasLimit:   event.GasLimit.Uint64(),
				Deadline:   event.Deadline.Uint64(),
				Target:     event.Target.String(),
				Calldata:   common.Bytes2Hex(event.Message),
				Layer1Hash: vLog.TxHash.Hex(),
			})
		case bridge_abi.L1RelayedMessageEventSignature:
			event := struct {
				MsgHash common.Hash
			}{}
			err := utils.UnpackLog(w.messengerABI, event, "RelayedMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer1 RelayedMessage event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}
			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MsgHash,
				txHash:       vLog.TxHash,
				isSuccessful: true,
			})
		case bridge_abi.L1FailedRelayedMessageEventSignature:
			event := struct {
				MsgHash common.Hash
			}{}
			err := utils.UnpackLog(w.messengerABI, event, "FailedRelayedMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer1 FailedRelayedMessage event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}
			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MsgHash,
				txHash:       vLog.TxHash,
				isSuccessful: false,
			})
		case bridge_abi.L1CommitBatchEventSignature:
			event := struct {
				BatchID    common.Hash
				BatchHash  common.Hash
				BatchIndex *big.Int
				ParentHash common.Hash
			}{}
			err := utils.UnpackLog(w.rollupABI, event, "CommitBatch", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer1 CommitBatch event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			rollupEvents = append(rollupEvents, rollupEvent{
				batchID: event.BatchID,
				txHash:  vLog.TxHash,
				status:  orm.RollupCommitted,
			})
		case bridge_abi.L1FinalizeBatchEventSignature:
			event := struct {
				BatchID    common.Hash
				BatchHash  common.Hash
				BatchIndex *big.Int
				ParentHash common.Hash
			}{}
			err := utils.UnpackLog(w.rollupABI, event, "FinalizeBatch", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer1 FinalizeBatch event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			rollupEvents = append(rollupEvents, rollupEvent{
				batchID: event.BatchID,
				txHash:  vLog.TxHash,
				status:  orm.RollupFinalized,
			})
		case bridge_abi.L1AppendMessageEventSignature:
			event := struct {
				MsgHash common.Hash
			}{}
			err := utils.UnpackLog(w.messageQueueABI, event, "AppendMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer1 AppendMessage event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}
			lastAppendMsgHash = event.MsgHash
		default:
			log.Error("Unknown event", "topic", vLog.Topics[0], "txHash", vLog.TxHash)
		}
	}

	return l1Messages, relayedMessages, rollupEvents, nil
}
