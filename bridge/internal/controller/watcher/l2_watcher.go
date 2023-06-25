package watcher

import (
	"context"
	"errors"
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

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"

	bridgeAbi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
	"scroll-tech/bridge/internal/utils"
)

// Metrics
var (
	bridgeL2MsgsSyncHeightGauge           = gethMetrics.NewRegisteredGauge("bridge/l2/msgs/sync/height", metrics.ScrollRegistry)
	bridgeL2BlocksFetchedHeightGauge      = gethMetrics.NewRegisteredGauge("bridge/l2/blocks/fetched/height", metrics.ScrollRegistry)
	bridgeL2BlocksFetchedGapGauge         = gethMetrics.NewRegisteredGauge("bridge/l2/blocks/fetched/gap", metrics.ScrollRegistry)
	bridgeL2MsgsSentEventsTotalCounter    = gethMetrics.NewRegisteredCounter("bridge/l2/msgs/sent/events/total", metrics.ScrollRegistry)
	bridgeL2MsgsAppendEventsTotalCounter  = gethMetrics.NewRegisteredCounter("bridge/l2/msgs/append/events/total", metrics.ScrollRegistry)
	bridgeL2MsgsRelayedEventsTotalCounter = gethMetrics.NewRegisteredCounter("bridge/l2/msgs/relayed/events/total", metrics.ScrollRegistry)
)

// L2WatcherClient provide APIs which support others to subscribe to various event from l2geth
type L2WatcherClient struct {
	ctx context.Context
	event.Feed

	*ethclient.Client

	db           *gorm.DB
	l2BlockOrm   *orm.L2Block
	chunkOrm     *orm.Chunk
	batchOrm     *orm.Batch
	l1MessageOrm *orm.L1Message
	l2MessageOrm *orm.L2Message

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
	l2MessageOrm := orm.NewL2Message(db)
	savedHeight, err := l2MessageOrm.GetLayer2LatestWatchedHeight()
	if err != nil {
		log.Warn("fetch height from db failed", "err", err)
		savedHeight = 0
	}

	w := L2WatcherClient{
		ctx:    ctx,
		db:     db,
		Client: client,

		l2BlockOrm:         orm.NewL2Block(db),
		chunkOrm:           orm.NewChunk(db),
		batchOrm:           orm.NewBatch(db),
		l1MessageOrm:       orm.NewL1Message(db),
		l2MessageOrm:       l2MessageOrm,
		processedMsgHeight: uint64(savedHeight),
		confirmations:      confirmations,

		messengerAddress: messengerAddress,
		messengerABI:     bridgeAbi.L2ScrollMessengerABI,

		messageQueueAddress:  messageQueueAddress,
		messageQueueABI:      bridgeAbi.L2MessageQueueABI,
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
	if count, err := w.batchOrm.GetBatchCount(w.ctx); err != nil {
		return fmt.Errorf("failed to get batch count: %v", err)
	} else if count > 0 {
		log.Info("genesis already imported", "batch count", count)
		return nil
	}

	genesis, err := w.HeaderByNumber(w.ctx, big.NewInt(0))
	if err != nil {
		return fmt.Errorf("failed to retrieve L2 genesis header: %v", err)
	}

	log.Info("retrieved L2 genesis header", "hash", genesis.Hash().String())

	chunk := &bridgeTypes.Chunk{
		Blocks: []*bridgeTypes.WrappedBlock{{
			Header:           genesis,
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		}},
	}

	chunkHash, err := chunk.Hash(0)
	if err != nil {
		return fmt.Errorf("failed to get L2 genesis chunk hash: %v", err)
	}

	batch, err := bridgeTypes.NewBatchHeader(0, 0, 0, common.Hash{}, []*bridgeTypes.Chunk{chunk})
	if err != nil {
		return fmt.Errorf("failed to get L2 genesis batch header: %v", err)
	}
	batchHash := batch.Hash().Hex()

	err = w.db.Transaction(func(dbTX *gorm.DB) error {
		if _, err = w.batchOrm.InsertBatch(w.ctx, 0, 0, chunkHash.Hex(), chunkHash.Hex(), []*bridgeTypes.Chunk{chunk}); err != nil {
			return fmt.Errorf("failed to insert batch: %v", err)
		}

		if err = w.chunkOrm.UpdateBatchHashInRange(w.ctx, 0, 0, batchHash, dbTX); err != nil {
			return fmt.Errorf("failed to update batch hash for L2 blocks: %v", err)
		}

		if _, err = w.chunkOrm.InsertChunk(w.ctx, chunk); err != nil {
			return fmt.Errorf("failed to insert chunk: %v", err)
		}

		if err = w.l2BlockOrm.UpdateChunkHashInRange(w.ctx, 0, 0, chunkHash.Hex(), dbTX); err != nil {
			log.Error("failed to update chunk_hash for l2_blocks",
				"chunk_hash", chunkHash, "start block", 0, "end block", 0, "err", err)
			return err
		}

		if err = w.chunkOrm.UpdateProvingStatus(w.ctx, chunkHash.Hex(), types.ProvingTaskVerified, dbTX); err != nil {
			return fmt.Errorf("failed to update genesis chunk proving status: %v", err)
		}

		if err = w.batchOrm.UpdateProvingStatus(w.ctx, batchHash, types.ProvingTaskVerified, dbTX); err != nil {
			return fmt.Errorf("failed to update genesis batch proving status: %v", err)
		}

		if err = w.batchOrm.UpdateRollupStatus(w.ctx, batchHash, types.RollupFinalized, dbTX); err != nil {
			return fmt.Errorf("failed to update genesis batch rollup status: %v", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("update genesis transaction failed: %v", err)
	}

	log.Info("successfully imported genesis chunk and batch")

	return nil
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
	for from := uint64(heightInDB) + 1; from <= blockHeight; from += blockTracesFetchLimit {
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
		txsData[i] = &gethTypes.TransactionData{
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
	var blocks []*bridgeTypes.WrappedBlock
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

		blocks = append(blocks, &bridgeTypes.WrappedBlock{
			Header:           block.Header(),
			Transactions:     txsToTxsData(block.Transactions()),
			WithdrawTrieRoot: common.BytesToHash(withdrawTrieRoot),
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
		query.Topics[0][0] = bridgeAbi.L2SentMessageEventSignature
		query.Topics[0][1] = bridgeAbi.L2RelayedMessageEventSignature
		query.Topics[0][2] = bridgeAbi.L2FailedRelayedMessageEventSignature
		query.Topics[0][3] = bridgeAbi.L2AppendMessageEventSignature

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
			if err = w.l1MessageOrm.UpdateLayer1StatusAndLayer2Hash(w.ctx, msg.msgHash.String(), msgStatus, msg.txHash.String()); err != nil {
				log.Error("Failed to update layer1 status and layer2 hash", "err", err)
				return
			}
		}

		if err = w.l2MessageOrm.SaveL2Messages(w.ctx, sentMessageEvents); err != nil {
			log.Error("failed to save l2 messages", "err", err)
			return
		}

		w.processedMsgHeight = uint64(to)
		bridgeL2MsgsSyncHeightGauge.Update(to)
	}
}

func (w *L2WatcherClient) parseBridgeEventLogs(logs []gethTypes.Log) ([]orm.L2Message, []relayedMessage, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l2Messages []orm.L2Message
	var relayedMessages []relayedMessage
	var lastAppendMsgHash common.Hash
	var lastAppendMsgNonce uint64
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case bridgeAbi.L2SentMessageEventSignature:
			event := bridgeAbi.L2SentMessageEvent{}
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

			l2Messages = append(l2Messages, orm.L2Message{
				Nonce:      event.MessageNonce.Uint64(),
				MsgHash:    computedMsgHash.String(),
				Height:     vLog.BlockNumber,
				Sender:     event.Sender.String(),
				Value:      event.Value.String(),
				Target:     event.Target.String(),
				Calldata:   common.Bytes2Hex(event.Message),
				Layer2Hash: vLog.TxHash.Hex(),
			})
		case bridgeAbi.L2RelayedMessageEventSignature:
			event := bridgeAbi.L2RelayedMessageEvent{}
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
		case bridgeAbi.L2FailedRelayedMessageEventSignature:
			event := bridgeAbi.L2FailedRelayedMessageEvent{}
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
		case bridgeAbi.L2AppendMessageEventSignature:
			event := bridgeAbi.L2AppendMessageEvent{}
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
