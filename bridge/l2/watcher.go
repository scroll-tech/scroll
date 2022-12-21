package l2

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/utils"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/bridge/config"
)

type relayedMessage struct {
	msgHash      common.Hash
	txHash       common.Hash
	isSuccessful bool
}

type importedBlock struct {
	blockHeight uint64
	blockHash   common.Hash
	txHash      common.Hash
}

// WatcherClient provide APIs which support others to subscribe to various event from l2geth
type WatcherClient struct {
	ctx context.Context
	event.Feed

	*ethclient.Client

	orm database.OrmFactory

	confirmations uint64

	messengerAddress common.Address
	messengerABI     *abi.ABI

	messageQueueAddress common.Address
	messageQueueABI     *abi.ABI

	blockContainerAddress common.Address
	blockContainerABI     *abi.ABI

	withdrawTrie *WithdrawTrie

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64

	stopped uint64
	stopCh  chan struct{}

	batchProposer *batchProposer
}

// NewL2WatcherClient take a l2geth instance to generate a l2watcherclient instance
func NewL2WatcherClient(ctx context.Context, client *ethclient.Client, confirmations uint64, bpCfg *config.BatchProposerConfig, messengerAddress, messageQueueAddress, blockContainerAddress common.Address, orm database.OrmFactory) *WatcherClient {
	savedHeight, err := orm.GetLayer2LatestWatchedHeight()
	if err != nil {
		log.Warn("fetch height from db failed", "err", err)
		savedHeight = 0
	}

	return &WatcherClient{
		ctx:                ctx,
		Client:             client,
		orm:                orm,
		processedMsgHeight: uint64(savedHeight),
		confirmations:      confirmations,

		messengerAddress: messengerAddress,
		messengerABI:     bridge_abi.L2MessengerABI,

		messageQueueAddress: messageQueueAddress,
		messageQueueABI:     bridge_abi.L2MessageQueueABI,

		blockContainerAddress: blockContainerAddress,
		blockContainerABI:     bridge_abi.L1BlockContainerABI,

		stopCh:        make(chan struct{}),
		stopped:       0,
		batchProposer: newBatchProposer(bpCfg, orm),
	}
}

// Start the Listening process
func (w *WatcherClient) Start() {
	go func() {
		if reflect.ValueOf(w.orm).IsNil() {
			panic("must run L2 watcher with DB")
		}

		lastFetchedBlock, err := w.orm.GetBlockTracesLatestHeight()
		if err != nil {
			panic(fmt.Sprintf("failed to GetBlockTracesLatestHeight in DB: %v", err))
		}

		if lastFetchedBlock < 0 {
			lastFetchedBlock = 0
		}
		lastBlockHeightChangeTime := time.Now()

		// trigger by timer
		// TODO: make it configurable
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// get current height
				number, err := w.BlockNumber(w.ctx)
				if err != nil {
					log.Error("failed to get_BlockNumber", "err", err)
					continue
				}
				duration := time.Since(lastBlockHeightChangeTime)
				var blockToFetch uint64
				if number > uint64(lastFetchedBlock)+w.confirmations {
					// latest block height changed
					blockToFetch = number - w.confirmations
				} else if duration.Seconds() > 60 {
					// l2geth didn't produce any blocks more than 1 minute.
					blockToFetch = number
				}
				// fetch at most `blockTracesFetchLimit=10` missing blocks
				if blockToFetch > uint64(lastFetchedBlock)+blockTracesFetchLimit {
					blockToFetch = uint64(lastFetchedBlock) + blockTracesFetchLimit
				}
				if lastFetchedBlock != int64(blockToFetch) {
					lastFetchedBlock = int64(blockToFetch)
					lastBlockHeightChangeTime = time.Now()
				}

				if err := w.tryFetchRunningMissingBlocks(w.ctx, blockToFetch); err != nil {
					log.Error("failed to fetchRunningMissingBlocks", "err", err)
				}

				// @todo handle error
				if err := w.fetchContractEvent(number); err != nil {
					log.Error("failed to fetchContractEvent", "err", err)
				}

				if err := w.batchProposer.tryProposeBatch(); err != nil {
					log.Error("failed to tryProposeBatch", "err", err)
				}

			case <-w.stopCh:
				return
			}
		}
	}()
}

// Stop the Watcher module, for a graceful shutdown.
func (w *WatcherClient) Stop() {
	w.stopCh <- struct{}{}
}

const blockTracesFetchLimit = uint64(10)

// try fetch missing blocks if inconsistent
func (w *WatcherClient) tryFetchRunningMissingBlocks(ctx context.Context, backTrackFrom uint64) error {
	// Get newest block in DB. must have blocks at that time.
	// Don't use "block_trace" table "trace" column's BlockTrace.Number,
	// because it might be empty if the corresponding rollup_result is finalized/finalization_skipped
	heightInDB, err := w.orm.GetBlockTracesLatestHeight()
	if err != nil {
		return fmt.Errorf("failed to GetBlockTracesLatestHeight in DB: %v", err)
	}
	backTrackTo := uint64(0)
	if heightInDB > 0 {
		backTrackTo = uint64(heightInDB)
	}

	// start backtracking

	var traces []*types.BlockTrace
	for number := backTrackFrom; number > backTrackTo; number-- {
		log.Debug("retrieving block trace", "height", number)
		trace, err2 := w.GetBlockTraceByNumber(ctx, big.NewInt(int64(number)))
		if err2 != nil {
			return fmt.Errorf("failed to GetBlockResultByHash: %v. number: %v", err2, number)
		}
		log.Info("retrieved block trace", "height", trace.Header.Number, "hash", trace.Header.Hash)

		traces = append(traces, trace)

	}
	if len(traces) > 0 {
		if err = w.orm.InsertBlockTraces(ctx, traces); err != nil {
			return fmt.Errorf("failed to batch insert BlockTraces: %v", err)
		}
	}
	return nil
}

const contractEventsBlocksFetchLimit = int64(10)

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (w *WatcherClient) fetchContractEvent(blockHeight uint64) error {
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
			w.blockContainerAddress,
		},
		Topics: make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 5)
	query.Topics[0][0] = bridge_abi.L2SendMessageEventSignature
	query.Topics[0][1] = bridge_abi.L2RelayedMessageEventSignature
	query.Topics[0][2] = bridge_abi.L2FailedRelayedMessageEventSignature
	query.Topics[0][3] = bridge_abi.L2AppendMessageEventSignature
	query.Topics[0][4] = bridge_abi.L2ImportBlockEventSignature

	logs, err := w.FilterLogs(w.ctx, query)
	if err != nil {
		log.Error("failed to get event logs", "err", err)
		return err
	}
	if len(logs) == 0 {
		w.processedMsgHeight = uint64(toBlock)
		return nil
	}
	log.Info("received new L2 messages", "fromBlock", fromBlock, "toBlock", toBlock,
		"cnt", len(logs))

	sentMessageEvents, relayedMessageEvents, importedBlockEvents, err := w.parseBridgeEventLogs(logs)
	if err != nil {
		log.Error("failed to parse emitted event log", "err", err)
		return err
	}

	// Update imported block first to make sure we don't forget to update importing blocks.
	for _, block := range importedBlockEvents {
		err := w.orm.UpdateL1BlockStatusAndImportTxHash(w.ctx, block.blockHash.String(), orm.L1BlockImported, block.txHash.String())
		if err != nil {
			log.Error("Failed to update l1 block status and import tx hash", "err", err)
			return err
		}
	}

	// Update relayed message first to make sure we don't forget to update submited message.
	// Since, we always start sync from the latest unprocessed message.
	for _, msg := range relayedMessageEvents {
		if msg.isSuccessful {
			// succeed
			err = w.orm.UpdateLayer1StatusAndLayer2Hash(w.ctx, msg.msgHash.String(), orm.MsgConfirmed, msg.txHash.String())
		} else {
			// failed
			err = w.orm.UpdateLayer1StatusAndLayer2Hash(w.ctx, msg.msgHash.String(), orm.MsgFailed, msg.txHash.String())
		}
		if err != nil {
			log.Error("Failed to update layer1 status and layer2 hash", "err", err)
			return err
		}
	}

	err = w.orm.SaveL2Messages(w.ctx, sentMessageEvents)
	if err == nil {
		w.processedMsgHeight = uint64(toBlock)
	}
	return err
}

func (w *WatcherClient) parseBridgeEventLogs(logs []types.Log) ([]*orm.L2Message, []relayedMessage, []importedBlock, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l2Messages []*orm.L2Message
	var relayedMessages []relayedMessage
	var importedBlocks []importedBlock
	var lastAppendMsgHash common.Hash
	var lastAppendMsgNonce uint64
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case bridge_abi.L2SendMessageEventSignature:
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
				log.Error("failed to unpack layer2 SentMessage event", "err", err)
				return l2Messages, relayedMessages, importedBlocks, err
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
			if event.MessageNonce.Uint64() != lastAppendMsgNonce {
				return l2Messages, relayedMessages, importedBlocks, errors.New("l2 message nonce mismatch")
			}
			if computedMsgHash != lastAppendMsgHash {
				return l2Messages, relayedMessages, importedBlocks, errors.New("l2 message hash mismatch")
			}
			l2Messages = append(l2Messages, &orm.L2Message{
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
				Layer2Hash: vLog.TxHash.Hex(),
			})
		case bridge_abi.L2RelayedMessageEventSignature:
			event := struct {
				MsgHash common.Hash
			}{}
			err := utils.UnpackLog(w.messengerABI, event, "RelayedMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer2 RelayedMessage event", "err", err)
				return l2Messages, relayedMessages, importedBlocks, err
			}
			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MsgHash,
				txHash:       vLog.TxHash,
				isSuccessful: true,
			})
		case bridge_abi.L2FailedRelayedMessageEventSignature:
			event := struct {
				MsgHash common.Hash
			}{}
			err := utils.UnpackLog(w.messengerABI, event, "FailedRelayedMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer2 FailedRelayedMessage event", "err", err)
				return l2Messages, relayedMessages, importedBlocks, err
			}
			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MsgHash,
				txHash:       vLog.TxHash,
				isSuccessful: false,
			})
		case bridge_abi.L2ImportBlockEventSignature:
			event := struct {
				BlockHash      common.Hash
				BlockHeight    *big.Int
				BlockTimestamp *big.Int
				StateRoot      common.Hash
			}{}
			err := utils.UnpackLog(w.blockContainerABI, event, "ImportBlock", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer2 ImportBlock event", "err", err)
				return l2Messages, relayedMessages, importedBlocks, err
			}
			importedBlocks = append(importedBlocks, importedBlock{
				blockHeight: event.BlockHeight.Uint64(),
				blockHash:   event.BlockHash,
				txHash:      vLog.TxHash,
			})
		case bridge_abi.L2AppendMessageEventSignature:
			event := struct {
				Index       *big.Int
				MessageHash common.Hash
			}{}
			err := utils.UnpackLog(w.messageQueueABI, event, "AppendMessage", vLog)
			if err != nil {
				log.Warn("Failed to unpack layer2 AppendMessage event", "err", err)
				return l2Messages, relayedMessages, importedBlocks, err
			}
			lastAppendMsgHash = event.MessageHash
			lastAppendMsgNonce = event.Index.Uint64()
		default:
			log.Error("Unknown event", "topic", vLog.Topics[0], "txHash", vLog.TxHash)
		}
	}

	return l2Messages, relayedMessages, importedBlocks, nil
}
