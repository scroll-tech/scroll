package watcher

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"

	geth "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	geth_metrics "github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	"scroll-tech/database"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/utils"
)

// Metrics
var (
	bridgeL2MsgsSyncHeightGauge      = geth_metrics.NewRegisteredGauge("bridge/l2/msgs/sync/height", metrics.ScrollRegistry)
	bridgeL2TracesFetchedHeightGauge = geth_metrics.NewRegisteredGauge("bridge/l2/traces/fetched/height", metrics.ScrollRegistry)
	bridgeL2TracesFetchedGapGauge    = geth_metrics.NewRegisteredGauge("bridge/l2/traces/fetched/gap", metrics.ScrollRegistry)

	bridgeL2MsgsSentEventsTotalCounter    = geth_metrics.NewRegisteredCounter("bridge/l2/msgs/sent/events/total", metrics.ScrollRegistry)
	bridgeL2MsgsAppendEventsTotalCounter  = geth_metrics.NewRegisteredCounter("bridge/l2/msgs/append/events/total", metrics.ScrollRegistry)
	bridgeL2MsgsRelayedEventsTotalCounter = geth_metrics.NewRegisteredCounter("bridge/l2/msgs/relayed/events/total", metrics.ScrollRegistry)
)

// L2WatcherClient provide APIs which support others to subscribe to various event from l2geth
type L2WatcherClient struct {
	ctx context.Context
	event.Feed

	*ethclient.Client

	orm database.OrmFactory

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
func NewL2WatcherClient(ctx context.Context, client *ethclient.Client, confirmations rpc.BlockNumber, messengerAddress, messageQueueAddress common.Address, withdrawTrieRootSlot common.Hash, orm database.OrmFactory) *L2WatcherClient {
	savedHeight, err := orm.GetLayer2LatestWatchedHeight()
	if err != nil {
		log.Warn("fetch height from db failed", "err", err)
		savedHeight = 0
	}

	w := L2WatcherClient{
		ctx:                ctx,
		Client:             client,
		orm:                orm,
		processedMsgHeight: uint64(savedHeight),
		confirmations:      confirmations,

		messengerAddress: messengerAddress,
		messengerABI:     bridge_abi.L2ScrollMessengerABI,

		messageQueueAddress:  messageQueueAddress,
		messageQueueABI:      bridge_abi.L2MessageQueueABI,
		withdrawTrieRootSlot: withdrawTrieRootSlot,

		stopped: 0,
	}

	// Initialize genesis before we do anything else
	if err := w.initializeGenesis(); err != nil {
		panic(fmt.Sprintf("failed to initialize L2 genesis batch, err: %v", err))
	}

	return &w
}

func (w *L2WatcherClient) initializeGenesis() error {
	if count, err := w.orm.GetBatchCount(); err != nil {
		return fmt.Errorf("failed to get batch count: %v", err)
	} else if count > 0 {
		log.Info("genesis already imported")
		return nil
	}

	genesis, err := w.HeaderByNumber(w.ctx, big.NewInt(0))
	if err != nil {
		return fmt.Errorf("failed to retrieve L2 genesis header: %v", err)
	}

	log.Info("retrieved L2 genesis header", "hash", genesis.Hash().String())

	blockTrace := &types.WrappedBlock{Header: genesis, Transactions: nil, WithdrawTrieRoot: common.Hash{}}
	batchData := types.NewGenesisBatchData(blockTrace)

	if err = AddBatchInfoToDB(w.orm, batchData, make([]*types.L2Message, 0), make([][]byte, 0)); err != nil {
		log.Error("failed to add batch info to DB", "BatchHash", batchData.Hash(), "error", err)
		return err
	}

	batchHash := batchData.Hash().Hex()

	if err = w.orm.UpdateProvingStatus(batchHash, types.ProvingTaskProved); err != nil {
		return fmt.Errorf("failed to update genesis batch proving status: %v", err)
	}

	if err = w.orm.UpdateRollupStatus(w.ctx, batchHash, types.RollupFinalized); err != nil {
		return fmt.Errorf("failed to update genesis batch rollup status: %v", err)
	}

	log.Info("successfully imported genesis batch")

	return nil
}

const blockTracesFetchLimit = uint64(10)

// TryFetchRunningMissingBlocks try fetch missing blocks if inconsistent
func (w *L2WatcherClient) TryFetchRunningMissingBlocks(ctx context.Context, blockHeight uint64) {
	// all messages should be fetched before blocks to make sure batch proposer work properly.
	processedMsgHeight := atomic.LoadUint64(&w.processedMsgHeight)
	if blockHeight > processedMsgHeight {
		blockHeight = processedMsgHeight
	}

	// Get newest block in DB. must have blocks at that time.
	// Don't use "block_trace" table "trace" column's BlockTrace.Number,
	// because it might be empty if the corresponding rollup_result is finalized/finalization_skipped
	heightInDB, err := w.orm.GetL2BlocksLatestHeight()
	if err != nil {
		log.Error("failed to GetL2BlocksLatestHeight", "err", err)
		return
	}

	// Can't get trace from genesis block, so the default start number is 1.
	var from = uint64(1)
	if heightInDB > 0 {
		from = uint64(heightInDB) + 1
	}

	for ; from <= blockHeight; from += blockTracesFetchLimit {
		to := from + blockTracesFetchLimit - 1

		if to > blockHeight {
			to = blockHeight
		}

		// Get block traces and insert into db.
		if err = w.getAndStoreBlockTraces(ctx, from, to); err != nil {
			log.Error("fail to getAndStoreBlockTraces", "from", from, "to", to, "err", err)
			return
		}
		bridgeL2TracesFetchedHeightGauge.Update(int64(to))
		bridgeL2TracesFetchedGapGauge.Update(int64(blockHeight - to))
	}
}

func txsToTxsData(txs geth_types.Transactions) []*geth_types.TransactionData {
	txsData := make([]*geth_types.TransactionData, len(txs))
	for i, tx := range txs {
		v, r, s := tx.RawSignatureValues()
		txsData[i] = &geth_types.TransactionData{
			Type:     tx.Type(),
			TxHash:   tx.Hash().String(),
			Nonce:    tx.Nonce(),
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
		block, err2 := w.BlockByNumber(ctx, big.NewInt(int64(number)))
		if err2 != nil {
			return fmt.Errorf("failed to GetBlockByNumber: %v. number: %v", err2, number)
		}

		log.Info("retrieved block", "height", block.Header().Number, "hash", block.Header().Hash().String())

		withdrawTrieRoot, err3 := w.StorageAt(ctx, w.messageQueueAddress, w.withdrawTrieRootSlot, big.NewInt(int64(number)))
		if err3 != nil {
			return fmt.Errorf("failed to get withdrawTrieRoot: %v. number: %v", err3, number)
		}

		blocks = append(blocks, &types.WrappedBlock{
			Header:           block.Header(),
			Transactions:     txsToTxsData(block.Transactions()),
			WithdrawTrieRoot: common.BytesToHash(withdrawTrieRoot),
		})
	}

	if len(blocks) > 0 {
		if err := w.orm.InsertWrappedBlocks(blocks); err != nil {
			return fmt.Errorf("failed to batch insert BlockTraces: %v", err)
		}
	}

	return nil
}

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (w *L2WatcherClient) FetchContractEvent() {
	defer func() {
		log.Info("l2 watcher fetchContractEvent", "w.processedMsgHeight", atomic.LoadUint64(&w.processedMsgHeight))
	}()

	blockHeight, err := utils.GetLatestConfirmedBlockNumber(w.ctx, w.Client, w.confirmations)
	if err != nil {
		log.Error("failed to get block number", "err", err)
		return
	}

	fromBlock := int64(atomic.LoadUint64(&w.processedMsgHeight)) + 1
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
		query.Topics[0][0] = bridge_abi.L2SentMessageEventSignature
		query.Topics[0][1] = bridge_abi.L2RelayedMessageEventSignature
		query.Topics[0][2] = bridge_abi.L2FailedRelayedMessageEventSignature
		query.Topics[0][3] = bridge_abi.L2AppendMessageEventSignature

		logs, err := w.FilterLogs(w.ctx, query)
		if err != nil {
			log.Error("failed to get event logs", "err", err)
			return
		}
		if len(logs) == 0 {
			atomic.StoreUint64(&w.processedMsgHeight, uint64(to))
			bridgeL2MsgsSyncHeightGauge.Update(to)
			continue
		}
		log.Info("received new L2 messages", "fromBlock", from, "toBlock", to, "cnt", len(logs))

		sentMessageEvents, relayedMessageEvents, err := w.parseBridgeEventLogs(logs)
		if err != nil {
			log.Error("failed to parse emitted event log", "err", err)
			return
		}

		sentMessageCount := int64(len(sentMessageEvents))
		relayedMessageCount := int64(len(relayedMessageEvents))
		bridgeL2MsgsSentEventsTotalCounter.Inc(sentMessageCount)
		bridgeL2MsgsRelayedEventsTotalCounter.Inc(relayedMessageCount)
		log.Info("L2 events types", "SentMessageCount", sentMessageCount, "RelayedMessageCount", relayedMessageCount)

		// Update relayed message first to make sure we don't forget to update submited message.
		// Since, we always start sync from the latest unprocessed message.
		for _, msg := range relayedMessageEvents {
			var msgStatus types.MsgStatus
			if msg.isSuccessful {
				msgStatus = types.MsgConfirmed
			} else {
				msgStatus = types.MsgFailed
			}
			if err = w.orm.UpdateLayer1StatusAndLayer2Hash(w.ctx, msg.msgHash.String(), msgStatus, msg.txHash.String()); err != nil {
				log.Error("Failed to update layer1 status and layer2 hash", "err", err)
				return
			}
		}

		if err = w.orm.SaveL2Messages(w.ctx, sentMessageEvents); err != nil {
			log.Error("failed to save l2 messages", "err", err)
			return
		}

		atomic.StoreUint64(&w.processedMsgHeight, uint64(to))
		bridgeL2MsgsSyncHeightGauge.Update(to)
	}
}

func (w *L2WatcherClient) parseBridgeEventLogs(logs []geth_types.Log) ([]*types.L2Message, []relayedMessage, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l2Messages []*types.L2Message
	var relayedMessages []relayedMessage
	var lastAppendMsgHash common.Hash
	var lastAppendMsgNonce uint64
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case bridge_abi.L2SentMessageEventSignature:
			event := bridge_abi.L2SentMessageEvent{}
			err := utils.UnpackLog(w.messengerABI, &event, "SentMessage", vLog)
			if err != nil {
				log.Error("failed to unpack layer2 SentMessage event", "err", err)
				return l2Messages, relayedMessages, err
			}

			computedMsgHash := utils.ComputeMessageHash(
				event.Sender,
				event.Target,
				event.Value,
				event.MessageNonce,
				event.Message,
			)

			// `AppendMessage` event is always emitted before `SentMessage` event
			// So they should always match, just double check
			if event.MessageNonce.Uint64() != lastAppendMsgNonce {
				errMsg := fmt.Sprintf("l2 message nonces mismatch: AppendMessage.nonce=%v, SentMessage.nonce=%v, tx_hash=%v",
					lastAppendMsgNonce, event.MessageNonce.Uint64(), vLog.TxHash.Hex())
				return l2Messages, relayedMessages, errors.New(errMsg)
			}
			if computedMsgHash != lastAppendMsgHash {
				errMsg := fmt.Sprintf("l2 message hashes mismatch: AppendMessage.msg_hash=%v, SentMessage.msg_hash=%v, tx_hash=%v",
					lastAppendMsgHash.Hex(), computedMsgHash.Hex(), vLog.TxHash.Hex())
				return l2Messages, relayedMessages, errors.New(errMsg)
			}

			l2Messages = append(l2Messages, &types.L2Message{
				Nonce:      event.MessageNonce.Uint64(),
				MsgHash:    computedMsgHash.String(),
				Height:     vLog.BlockNumber,
				Sender:     event.Sender.String(),
				Value:      event.Value.String(),
				Target:     event.Target.String(),
				Calldata:   common.Bytes2Hex(event.Message),
				Layer2Hash: vLog.TxHash.Hex(),
			})
		case bridge_abi.L2RelayedMessageEventSignature:
			event := bridge_abi.L2RelayedMessageEvent{}
			err := utils.UnpackLog(w.messengerABI, &event, "RelayedMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer2 RelayedMessage event", "err", err)
				return l2Messages, relayedMessages, err
			}

			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MessageHash,
				txHash:       vLog.TxHash,
				isSuccessful: true,
			})
		case bridge_abi.L2FailedRelayedMessageEventSignature:
			event := bridge_abi.L2FailedRelayedMessageEvent{}
			err := utils.UnpackLog(w.messengerABI, &event, "FailedRelayedMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer2 FailedRelayedMessage event", "err", err)
				return l2Messages, relayedMessages, err
			}

			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MessageHash,
				txHash:       vLog.TxHash,
				isSuccessful: false,
			})
		case bridge_abi.L2AppendMessageEventSignature:
			event := bridge_abi.L2AppendMessageEvent{}
			err := utils.UnpackLog(w.messageQueueABI, &event, "AppendMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer2 AppendMessage event", "err", err)
				return l2Messages, relayedMessages, err
			}

			lastAppendMsgHash = event.MessageHash
			lastAppendMsgNonce = event.Index.Uint64()
			bridgeL2MsgsAppendEventsTotalCounter.Inc(1)
		default:
			log.Error("Unknown event", "topic", vLog.Topics[0], "txHash", vLog.TxHash)
		}
	}

	return l2Messages, relayedMessages, nil
}
