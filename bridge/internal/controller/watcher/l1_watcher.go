package watcher

import (
	"context"
	"math/big"

	geth "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"

	bridgeAbi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/internal/orm"
	"scroll-tech/bridge/internal/utils"
)

var (
	bridgeL1MsgsSyncHeightGauge           = gethMetrics.NewRegisteredGauge("bridge/l1/msgs/sync/height", metrics.ScrollRegistry)
	bridgeL1MsgsSentEventsTotalCounter    = gethMetrics.NewRegisteredCounter("bridge/l1/msgs/sent/events/total", metrics.ScrollRegistry)
	bridgeL1MsgsRelayedEventsTotalCounter = gethMetrics.NewRegisteredCounter("bridge/l1/msgs/relayed/events/total", metrics.ScrollRegistry)
	bridgeL1MsgsRollupEventsTotalCounter  = gethMetrics.NewRegisteredCounter("bridge/l1/msgs/rollup/events/total", metrics.ScrollRegistry)
)

type rollupEvent struct {
	batchHash common.Hash
	txHash    common.Hash
	status    types.RollupStatus
}

// L1WatcherClient will listen for smart contract events from Eth L1.
type L1WatcherClient struct {
	ctx          context.Context
	client       *ethclient.Client
	l1MessageOrm *orm.L1Message
	l2MessageOrm *orm.L2Message
	l1BlockOrm   *orm.L1Block
	l1BatchOrm   *orm.Batch

	// The number of new blocks to wait for a block to be confirmed
	confirmations rpc.BlockNumber

	messengerAddress common.Address
	messengerABI     *abi.ABI

	messageQueueAddress common.Address
	messageQueueABI     *abi.ABI

	scrollChainAddress common.Address
	scrollChainABI     *abi.ABI

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64
	// The height of the block that the watcher has retrieved header rlp
	processedBlockHeight uint64
}

// NewL1WatcherClient returns a new instance of L1WatcherClient.
func NewL1WatcherClient(ctx context.Context, client *ethclient.Client, startHeight uint64, confirmations rpc.BlockNumber, messengerAddress, messageQueueAddress, scrollChainAddress common.Address, db *gorm.DB) *L1WatcherClient {
	l1MessageOrm := orm.NewL1Message(db)
	savedHeight, err := l1MessageOrm.GetLayer1LatestWatchedHeight()
	if err != nil {
		log.Warn("Failed to fetch height from db", "err", err)
		savedHeight = 0
	}
	if savedHeight < int64(startHeight) {
		savedHeight = int64(startHeight)
	}

	l1BlockOrm := orm.NewL1Block(db)
	savedL1BlockHeight, err := l1BlockOrm.GetLatestL1BlockHeight()
	if err != nil {
		log.Warn("Failed to fetch latest L1 block height from db", "err", err)
		savedL1BlockHeight = 0
	}
	if savedL1BlockHeight < startHeight {
		savedL1BlockHeight = startHeight
	}

	return &L1WatcherClient{
		ctx:           ctx,
		client:        client,
		l1MessageOrm:  l1MessageOrm,
		l1BlockOrm:    l1BlockOrm,
		l1BatchOrm:    orm.NewBatch(db),
		l2MessageOrm:  orm.NewL2Message(db),
		confirmations: confirmations,

		messengerAddress: messengerAddress,
		messengerABI:     bridgeAbi.L1ScrollMessengerABI,

		messageQueueAddress: messageQueueAddress,
		messageQueueABI:     bridgeAbi.L1MessageQueueABI,

		scrollChainAddress: scrollChainAddress,
		scrollChainABI:     bridgeAbi.ScrollChainABI,

		processedMsgHeight:   uint64(savedHeight),
		processedBlockHeight: savedL1BlockHeight,
	}
}

// ProcessedBlockHeight get processedBlockHeight
// Currently only use for unit test
func (w *L1WatcherClient) ProcessedBlockHeight() uint64 {
	return w.processedBlockHeight
}

// Confirmations get confirmations
// Currently only use for unit test
func (w *L1WatcherClient) Confirmations() rpc.BlockNumber {
	return w.confirmations
}

// SetConfirmations set the confirmations for L1WatcherClient
// Currently only use for unit test
func (w *L1WatcherClient) SetConfirmations(confirmations rpc.BlockNumber) {
	w.confirmations = confirmations
}

// FetchBlockHeader pull latest L1 blocks and save in DB
func (w *L1WatcherClient) FetchBlockHeader(blockHeight uint64) error {
	fromBlock := int64(w.processedBlockHeight) + 1
	toBlock := int64(blockHeight)
	if toBlock < fromBlock {
		return nil
	}
	if toBlock > fromBlock+contractEventsBlocksFetchLimit {
		toBlock = fromBlock + contractEventsBlocksFetchLimit - 1
	}

	var blocks []orm.L1Block
	var err error
	height := fromBlock
	for ; height <= toBlock; height++ {
		var block *gethTypes.Header
		block, err = w.client.HeaderByNumber(w.ctx, big.NewInt(height))
		if err != nil {
			log.Warn("Failed to get block", "height", height, "err", err)
			break
		}
		blocks = append(blocks, orm.L1Block{
			Number: uint64(height),
			Hash:   block.Hash().String(),
		})
	}

	// failed at first block, return with the error
	if height == fromBlock {
		return err
	}
	toBlock = height - 1

	// insert succeed blocks
	err = w.l1BlockOrm.InsertL1Blocks(w.ctx, blocks)
	if err != nil {
		log.Warn("Failed to insert L1 block to db", "fromBlock", fromBlock, "toBlock", toBlock, "err", err)
		return err
	}

	// update processed height
	w.processedBlockHeight = uint64(toBlock)
	return nil
}

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (w *L1WatcherClient) FetchContractEvent() error {
	defer func() {
		log.Info("l1 watcher fetchContractEvent", "w.processedMsgHeight", w.processedMsgHeight)
	}()
	blockHeight, err := utils.GetLatestConfirmedBlockNumber(w.ctx, w.client, w.confirmations)
	if err != nil {
		log.Error("failed to get block number", "err", err)
		return err
	}

	fromBlock := int64(w.processedMsgHeight) + 1
	toBlock := int64(blockHeight)

	for from := fromBlock; from <= toBlock; from += contractEventsBlocksFetchLimit {
		to := from + contractEventsBlocksFetchLimit - 1

		if to > toBlock {
			to = toBlock
		}

		// warning: uint int conversion...
		query := geth.FilterQuery{
			FromBlock: big.NewInt(from), // inclusive
			ToBlock:   big.NewInt(to),   // inclusive
			Addresses: []common.Address{
				w.messengerAddress,
				w.scrollChainAddress,
				w.messageQueueAddress,
			},
			Topics: make([][]common.Hash, 1),
		}
		query.Topics[0] = make([]common.Hash, 5)
		query.Topics[0][0] = bridgeAbi.L1QueueTransactionEventSignature
		query.Topics[0][1] = bridgeAbi.L1RelayedMessageEventSignature
		query.Topics[0][2] = bridgeAbi.L1FailedRelayedMessageEventSignature
		query.Topics[0][3] = bridgeAbi.L1CommitBatchEventSignature
		query.Topics[0][4] = bridgeAbi.L1FinalizeBatchEventSignature

		logs, err := w.client.FilterLogs(w.ctx, query)
		if err != nil {
			log.Warn("Failed to get event logs", "err", err)
			return err
		}
		if len(logs) == 0 {
			w.processedMsgHeight = uint64(to)
			bridgeL1MsgsSyncHeightGauge.Update(to)
			continue
		}
		log.Info("Received new L1 events", "fromBlock", from, "toBlock", to, "cnt", len(logs))

		sentMessageEvents, relayedMessageEvents, rollupEvents, err := w.parseBridgeEventLogs(logs)
		if err != nil {
			log.Error("Failed to parse emitted events log", "err", err)
			return err
		}
		sentMessageCount := int64(len(sentMessageEvents))
		relayedMessageCount := int64(len(relayedMessageEvents))
		rollupEventCount := int64(len(rollupEvents))
		bridgeL1MsgsSentEventsTotalCounter.Inc(sentMessageCount)
		bridgeL1MsgsRelayedEventsTotalCounter.Inc(relayedMessageCount)
		bridgeL1MsgsRollupEventsTotalCounter.Inc(rollupEventCount)
		log.Info("L1 events types", "SentMessageCount", sentMessageCount, "RelayedMessageCount", relayedMessageCount, "RollupEventCount", rollupEventCount)

		// use rollup event to update rollup results db status
		var batchHashes []string
		for _, event := range rollupEvents {
			batchHashes = append(batchHashes, event.batchHash.String())
		}
		statuses, err := w.l1BatchOrm.GetRollupStatusByHashList(w.ctx, batchHashes)
		if err != nil {
			log.Error("Failed to GetRollupStatusByHashList", "err", err)
			return err
		}
		if len(statuses) != len(batchHashes) {
			log.Error("RollupStatus.Length mismatch with batchHashes.Length", "RollupStatus.Length", len(statuses), "batchHashes.Length", len(batchHashes))
			return nil
		}

		for index, event := range rollupEvents {
			batchHash := event.batchHash.String()
			status := statuses[index]
			// only update when db status is before event status
			if event.status > status {
				if event.status == types.RollupFinalized {
					updateFields := map[string]interface{}{
						"finalize_tx_hash": event.txHash.String(),
						"rollup_status":    types.RollupFinalized,
					}
					err = w.l1BatchOrm.UpdateChunkBatch(w.ctx, batchHash, updateFields)
				} else if event.status == types.RollupCommitted {
					updateFields := map[string]interface{}{
						"commit_tx_hash": event.txHash.String(),
						"rollup_status":  types.RollupCommitted,
					}
					err = w.l1BatchOrm.UpdateChunkBatch(w.ctx, batchHash, updateFields)
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
			var msgStatus types.MsgStatus
			if msg.isSuccessful {
				msgStatus = types.MsgConfirmed
			} else {
				msgStatus = types.MsgFailed
			}
			if err = w.l2MessageOrm.UpdateLayer2StatusAndLayer1Hash(w.ctx, msg.msgHash.String(), msgStatus, msg.txHash.String()); err != nil {
				log.Error("Failed to update layer1 status and layer2 hash", "err", err)
				return err
			}
		}

		if err = w.l1MessageOrm.SaveL1Messages(w.ctx, sentMessageEvents); err != nil {
			return err
		}

		w.processedMsgHeight = uint64(to)
		bridgeL1MsgsSyncHeightGauge.Update(to)
	}

	return nil
}

func (w *L1WatcherClient) parseBridgeEventLogs(logs []gethTypes.Log) ([]*orm.L1Message, []relayedMessage, []rollupEvent, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up
	var l1Messages []*orm.L1Message
	var relayedMessages []relayedMessage
	var rollupEvents []rollupEvent
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case bridgeAbi.L1QueueTransactionEventSignature:
			event := bridgeAbi.L1QueueTransactionEvent{}
			err := utils.UnpackLog(w.messageQueueABI, &event, "QueueTransaction", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer1 QueueTransaction event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			msgHash := common.BytesToHash(crypto.Keccak256(event.Data))

			l1Messages = append(l1Messages, &orm.L1Message{
				QueueIndex: event.QueueIndex.Uint64(),
				MsgHash:    msgHash.String(),
				Height:     vLog.BlockNumber,
				Sender:     event.Sender.String(),
				Value:      event.Value.String(),
				Target:     event.Target.String(),
				Calldata:   common.Bytes2Hex(event.Data),
				GasLimit:   event.GasLimit.Uint64(),
				Layer1Hash: vLog.TxHash.Hex(),
			})
		case bridgeAbi.L1RelayedMessageEventSignature:
			event := bridgeAbi.L1RelayedMessageEvent{}
			err := utils.UnpackLog(w.messengerABI, &event, "RelayedMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer1 RelayedMessage event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MessageHash,
				txHash:       vLog.TxHash,
				isSuccessful: true,
			})
		case bridgeAbi.L1FailedRelayedMessageEventSignature:
			event := bridgeAbi.L1FailedRelayedMessageEvent{}
			err := utils.UnpackLog(w.messengerABI, &event, "FailedRelayedMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer1 FailedRelayedMessage event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MessageHash,
				txHash:       vLog.TxHash,
				isSuccessful: false,
			})
		case bridgeAbi.L1CommitBatchEventSignature:
			event := bridgeAbi.L1CommitBatchEvent{}
			err := utils.UnpackLog(w.scrollChainABI, &event, "CommitBatch", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer1 CommitBatch event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			rollupEvents = append(rollupEvents, rollupEvent{
				batchHash: event.BatchHash,
				txHash:    vLog.TxHash,
				status:    types.RollupCommitted,
			})
		case bridgeAbi.L1FinalizeBatchEventSignature:
			event := bridgeAbi.L1FinalizeBatchEvent{}
			err := utils.UnpackLog(w.scrollChainABI, &event, "FinalizeBatch", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer1 FinalizeBatch event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			rollupEvents = append(rollupEvents, rollupEvent{
				batchHash: event.BatchHash,
				txHash:    vLog.TxHash,
				status:    types.RollupFinalized,
			})
		default:
			log.Error("Unknown event", "topic", vLog.Topics[0], "txHash", vLog.TxHash)
		}
	}

	return l1Messages, relayedMessages, rollupEvents, nil
}
