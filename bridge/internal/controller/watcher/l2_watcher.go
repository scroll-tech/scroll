package watcher

import (
	"context"
	"fmt"
	"math/big"

	geth "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	"scroll-tech/common/bytecode/scroll/L2"
	"scroll-tech/common/bytecode/scroll/L2/predeploys"
	"scroll-tech/common/metrics"
	"scroll-tech/common/types"

	"scroll-tech/bridge/internal/orm"
	"scroll-tech/bridge/internal/utils"
)

// Metrics
var (
	bridgeL2MsgsSyncHeightGauge           = gethMetrics.NewRegisteredGauge("bridge/l2/msgs/sync/height", metrics.ScrollRegistry)
	bridgeL2BlocksFetchedHeightGauge      = gethMetrics.NewRegisteredGauge("bridge/l2/blocks/fetched/height", metrics.ScrollRegistry)
	bridgeL2BlocksFetchedGapGauge         = gethMetrics.NewRegisteredGauge("bridge/l2/blocks/fetched/gap", metrics.ScrollRegistry)
	bridgeL2MsgsRelayedEventsTotalCounter = gethMetrics.NewRegisteredCounter("bridge/l2/msgs/relayed/events/total", metrics.ScrollRegistry)
)

// L2WatcherClient provide APIs which support others to subscribe to various event from l2geth
type L2WatcherClient struct {
	ctx context.Context
	event.Feed

	*ethclient.Client

	l2BlockOrm   *orm.L2Block
	l1MessageOrm *orm.L1Message

	confirmations rpc.BlockNumber

	messengerAddress common.Address
	messengerABI     *abi.ABI

	messageQueueAddress  common.Address
	messageQueueABI      *abi.ABI
	withdrawTrieRootSlot common.Hash

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64

	stopped uint64
}

// NewL2WatcherClient take a l2geth instance to generate a l2watcherclient instance
func NewL2WatcherClient(ctx context.Context, client *ethclient.Client, confirmations rpc.BlockNumber, messengerAddress, messageQueueAddress common.Address, withdrawTrieRootSlot common.Hash, db *gorm.DB) *L2WatcherClient {
	l1MessageOrm := orm.NewL1Message(db)
	var savedHeight uint64
	l1msg, err := l1MessageOrm.GetLayer1LatestMessageWithLayer2Hash()
	if err != nil || l1msg == nil {
		log.Warn("fetch height from db failed", "err", err)
		savedHeight = 0
	} else {
		receipt, err := client.TransactionReceipt(ctx, common.HexToHash(l1msg.Layer2Hash))
		if err != nil || receipt == nil {
			log.Warn("get tx from l2 failed", "err", err)
			savedHeight = 0
		} else {
			savedHeight = receipt.BlockNumber.Uint64()
		}
	}

	w := L2WatcherClient{
		ctx:    ctx,
		Client: client,

		l2BlockOrm:         orm.NewL2Block(db),
		l1MessageOrm:       orm.NewL1Message(db),
		processedMsgHeight: savedHeight,
		confirmations:      confirmations,

		messengerAddress: messengerAddress,
		messengerABI:     L2.L2ScrollMessengerABI,

		messageQueueAddress:  messageQueueAddress,
		messageQueueABI:      predeploys.L2MessageQueueABI,
		withdrawTrieRootSlot: withdrawTrieRootSlot,

		stopped: 0,
	}

	return &w
}

const blockTracesFetchLimit = uint64(10)

// TryFetchRunningMissingBlocks attempts to fetch and store block traces for any missing blocks.
func (w *L2WatcherClient) TryFetchRunningMissingBlocks(blockHeight uint64) {
	heightInDB, err := w.l2BlockOrm.GetL2BlocksLatestHeight(w.ctx)
	if err != nil {
		log.Error("failed to GetL2BlocksLatestHeight", "err", err)
		return
	}

	// Fetch and store block traces for missing blocks
	for from := heightInDB + 1; from <= blockHeight; from += blockTracesFetchLimit {
		to := from + blockTracesFetchLimit - 1

		if to > blockHeight {
			to = blockHeight
		}

		if err = w.getAndStoreBlockTraces(w.ctx, from, to); err != nil {
			log.Error("fail to getAndStoreBlockTraces", "from", from, "to", to, "err", err)
			return
		}
		bridgeL2BlocksFetchedHeightGauge.Update(int64(to))
		bridgeL2BlocksFetchedGapGauge.Update(int64(blockHeight - to))
	}
}

func txsToTxsData(txs gethTypes.Transactions) []*gethTypes.TransactionData {
	txsData := make([]*gethTypes.TransactionData, len(txs))
	for i, tx := range txs {
		v, r, s := tx.RawSignatureValues()

		nonce := tx.Nonce()

		// We need QueueIndex in `NewBatchHeader`. However, `TransactionData`
		// does not have this field. Since `L1MessageTx` do not have a nonce,
		// we reuse this field for storing the queue index.
		if msg := tx.AsL1MessageTx(); msg != nil {
			nonce = msg.QueueIndex
		}

		txsData[i] = &gethTypes.TransactionData{
			Type:     tx.Type(),
			TxHash:   tx.Hash().String(),
			Nonce:    nonce,
			ChainId:  (*hexutil.Big)(tx.ChainId()),
			Gas:      tx.Gas(),
			GasPrice: (*hexutil.Big)(tx.GasPrice()),
			To:       tx.To(),
			Value:    (*hexutil.Big)(tx.Value()),
			Data:     hexutil.Encode(tx.Data()),
			IsCreate: tx.To() == nil,
			V:        (*hexutil.Big)(v),
			R:        (*hexutil.Big)(r),
			S:        (*hexutil.Big)(s),
		}
	}
	return txsData
}

func (w *L2WatcherClient) getAndStoreBlockTraces(ctx context.Context, from, to uint64) error {
	var blocks []*types.WrappedBlock
	for number := from; number <= to; number++ {
		log.Debug("retrieving block", "height", number)
		block, err := w.GetBlockByNumberOrHash(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(number)))
		if err != nil {
			return fmt.Errorf("failed to GetBlockByNumberOrHash: %v. number: %v", err, number)
		}
		if block.RowConsumption == nil {
			return fmt.Errorf("fetched block does not contain RowConsumption. number: %v", number)
		}

		log.Info("retrieved block", "height", block.Header().Number, "hash", block.Header().Hash().String())

		withdrawRoot, err3 := w.StorageAt(ctx, w.messageQueueAddress, w.withdrawTrieRootSlot, big.NewInt(int64(number)))
		if err3 != nil {
			return fmt.Errorf("failed to get withdrawRoot: %v. number: %v", err3, number)
		}
		blocks = append(blocks, &types.WrappedBlock{
			Header:         block.Header(),
			Transactions:   txsToTxsData(block.Transactions()),
			WithdrawRoot:   common.BytesToHash(withdrawRoot),
			RowConsumption: block.RowConsumption,
		})
	}

	if len(blocks) > 0 {
		if err := w.l2BlockOrm.InsertL2Blocks(w.ctx, blocks); err != nil {
			return fmt.Errorf("failed to batch insert BlockTraces: %v", err)
		}
	}

	return nil
}

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (w *L2WatcherClient) FetchContractEvent() {
	defer func() {
		log.Info("l2 watcher fetchContractEvent", "w.processedMsgHeight", w.processedMsgHeight)
	}()

	blockHeight, err := utils.GetLatestConfirmedBlockNumber(w.ctx, w.Client, w.confirmations)
	if err != nil {
		log.Error("failed to get block number", "err", err)
		return
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
				w.messageQueueAddress,
			},
			Topics: make([][]common.Hash, 1),
		}
		query.Topics[0] = make([]common.Hash, 4)
		query.Topics[0][0] = L2.L2ScrollMessengerSentMessageEventSignature
		query.Topics[0][1] = L2.L2ScrollMessengerRelayedMessageEventSignature
		query.Topics[0][2] = L2.L2ScrollMessengerFailedRelayedMessageEventSignature
		query.Topics[0][3] = predeploys.L2MessageQueueAppendMessageEventSignature

		logs, err := w.FilterLogs(w.ctx, query)
		if err != nil {
			log.Error("failed to get event logs", "err", err)
			return
		}
		if len(logs) == 0 {
			w.processedMsgHeight = uint64(to)
			bridgeL2MsgsSyncHeightGauge.Update(to)
			continue
		}
		log.Info("received new L2 messages", "fromBlock", from, "toBlock", to, "cnt", len(logs))

		relayedMessageEvents, err := w.parseBridgeEventLogs(logs)
		if err != nil {
			log.Error("failed to parse emitted event log", "err", err)
			return
		}

		relayedMessageCount := int64(len(relayedMessageEvents))
		bridgeL2MsgsRelayedEventsTotalCounter.Inc(relayedMessageCount)
		log.Info("L2 events types", "RelayedMessageCount", relayedMessageCount)

		// Update relayed message first to make sure we don't forget to update submited message.
		// Since, we always start sync from the latest unprocessed message.
		for _, msg := range relayedMessageEvents {
			var msgStatus types.MsgStatus
			if msg.isSuccessful {
				msgStatus = types.MsgConfirmed
			} else {
				msgStatus = types.MsgFailed
			}
			if err = w.l1MessageOrm.UpdateLayer1StatusAndLayer2Hash(w.ctx, msg.msgHash.String(), msgStatus, msg.txHash.String()); err != nil {
				log.Error("Failed to update layer1 status and layer2 hash", "err", err)
				return
			}
		}

		w.processedMsgHeight = uint64(to)
		bridgeL2MsgsSyncHeightGauge.Update(to)
	}
}

func (w *L2WatcherClient) parseBridgeEventLogs(logs []gethTypes.Log) ([]relayedMessage, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var relayedMessages []relayedMessage
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case L2.L2ScrollMessengerRelayedMessageEventSignature:
			event := L2.L2ScrollMessengerRelayedMessageEvent{}
			err := utils.UnpackLog(w.messengerABI, &event, "RelayedMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer2 RelayedMessage event", "err", err)
				return relayedMessages, err
			}

			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MessageHash,
				txHash:       vLog.TxHash,
				isSuccessful: true,
			})
		case L2.L2ScrollMessengerFailedRelayedMessageEventSignature:
			event := L2.L2ScrollMessengerFailedRelayedMessageEvent{}
			err := utils.UnpackLog(w.messengerABI, &event, "FailedRelayedMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer2 FailedRelayedMessage event", "err", err)
				return relayedMessages, err
			}

			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MessageHash,
				txHash:       vLog.TxHash,
				isSuccessful: false,
			})
			log.Error("Unknown event", "topic", vLog.Topics[0], "txHash", vLog.TxHash)
		}
	}

	return relayedMessages, nil
}
