package l2

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"time"

	geth "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/utils"

	"scroll-tech/common/viper"
	"scroll-tech/database"
	"scroll-tech/database/orm"
)

type relayedMessage struct {
	msgHash      common.Hash
	txHash       common.Hash
	isSuccessful bool
}

// WatcherClient provide APIs which support others to subscribe to various event from l2geth
type WatcherClient struct {
	ctx context.Context
	event.Feed

	*ethclient.Client

	orm database.OrmFactory

	vp               *viper.Viper
	messengerAddress common.Address
	messengerABI     *abi.ABI

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64

	stopped uint64
	stopCh  chan struct{}

	batchProposer *batchProposer
}

// NewL2WatcherClient take a l2geth instance to generate a l2watcherclient instance
func NewL2WatcherClient(ctx context.Context, client *ethclient.Client, vp *viper.Viper, orm database.OrmFactory) *WatcherClient {
	savedHeight, err := orm.GetLayer2LatestWatchedHeight()
	if err != nil {
		log.Warn("fetch height from db failed", "err", err)
		savedHeight = 0
	}

	messengerAddress := vp.GetAddress("l2_messenger_address")

	return &WatcherClient{
		ctx:                ctx,
		Client:             client,
		orm:                orm,
		processedMsgHeight: uint64(savedHeight),
		vp:                 vp,
		messengerAddress:   messengerAddress,
		messengerABI:       bridge_abi.L2MessengerMetaABI,
		stopCh:             make(chan struct{}),
		stopped:            0,
		batchProposer:      newBatchProposer(vp.Sub("batch_proposer_config"), orm),
	}
}

// Start the Listening process
func (w *WatcherClient) Start() {
	go func() {
		if reflect.ValueOf(w.orm).IsNil() {
			panic("must run L2 watcher with DB")
		}

		// trigger by timer
		watcherTimeSec := w.vp.GetInt("watcher_time_sec")
		ticker := time.NewTicker(time.Duration(watcherTimeSec) * time.Second)
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

				confirmations := uint64(w.vp.GetInt64("confirmations"))
				if number >= confirmations {
					number = number - confirmations
				} else {
					number = 0
				}

				if err := w.tryFetchRunningMissingBlocks(w.ctx, number); err != nil {
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

	// note that backTrackFrom >= backTrackTo because we are doing backtracking
	blockTracesFetchLimit := uint64(w.vp.GetInt64("block_traces_fetch_limit"))
	if backTrackFrom > backTrackTo+blockTracesFetchLimit {
		backTrackFrom = backTrackTo + blockTracesFetchLimit
	}

	// start backtracking

	var traces []*types.BlockTrace
	for number := backTrackFrom; number > backTrackTo; number-- {
		log.Debug("retrieving block trace", "height", number)
		trace, err2 := w.GetBlockTraceByNumber(ctx, big.NewInt(int64(number)))
		if err2 != nil {
			return fmt.Errorf("failed to GetBlockResultByHash: %v. number: %v", err2, number)
		}
		log.Info("retrieved block trace", "height", trace.Header.Number, "hash", trace.Header.Hash().String())

		traces = append(traces, trace)

	}
	if len(traces) > 0 {
		if err = w.orm.InsertBlockTraces(traces); err != nil {
			return fmt.Errorf("failed to batch insert BlockTraces: %v", err)
		}
	}
	return nil
}

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (w *WatcherClient) fetchContractEvent(blockHeight uint64) error {
	fromBlock := int64(w.processedMsgHeight) + 1
	toBlock := int64(blockHeight)

	if toBlock < fromBlock {
		return nil
	}

	contractEventsBlocksFetchLimit := w.vp.GetInt64("contract_events_blocks_fetch_limit")
	if toBlock > fromBlock+contractEventsBlocksFetchLimit {
		toBlock = fromBlock + contractEventsBlocksFetchLimit - 1
	}

	// warning: uint int conversion...
	query := geth.FilterQuery{
		FromBlock: big.NewInt(fromBlock), // inclusive
		ToBlock:   big.NewInt(toBlock),   // inclusive
		Addresses: []common.Address{
			w.messengerAddress,
		},
		Topics: make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 3)
	query.Topics[0][0] = common.HexToHash(bridge_abi.SENT_MESSAGE_EVENT_SIGNATURE)
	query.Topics[0][1] = common.HexToHash(bridge_abi.RELAYED_MESSAGE_EVENT_SIGNATURE)
	query.Topics[0][2] = common.HexToHash(bridge_abi.FAILED_RELAYED_MESSAGE_EVENT_SIGNATURE)

	logs, err := w.FilterLogs(w.ctx, query)
	if err != nil {
		log.Error("failed to get event logs", "err", err)
		return err
	}
	if len(logs) == 0 {
		w.processedMsgHeight = uint64(toBlock)
		log.Info("l2 watcher fetchContractEvent", "w.processedMsgHeight", w.processedMsgHeight)
		return nil
	}
	log.Info("received new L2 messages", "fromBlock", fromBlock, "toBlock", toBlock,
		"cnt", len(logs))

	sentMessageEvents, relayedMessageEvents, err := w.parseBridgeEventLogs(logs)
	if err != nil {
		log.Error("failed to parse emitted event log", "err", err)
		return err
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
		log.Info("l2 watcher fetchContractEvent", "w.processedMsgHeight", w.processedMsgHeight)
	}
	return err
}

func (w *WatcherClient) parseBridgeEventLogs(logs []types.Log) ([]*orm.L2Message, []relayedMessage, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l2Messages []*orm.L2Message
	var relayedMessages []relayedMessage
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case common.HexToHash(bridge_abi.SENT_MESSAGE_EVENT_SIGNATURE):
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

			err := w.messengerABI.UnpackIntoInterface(&event, "SentMessage", vLog.Data)
			if err != nil {
				log.Error("failed to unpack layer2 SentMessage event", "err", err)
				return l2Messages, relayedMessages, err
			}
			// target is in topics[1]
			event.Target = common.HexToAddress(vLog.Topics[1].String())
			l2Messages = append(l2Messages, &orm.L2Message{
				Nonce:      event.MessageNonce.Uint64(),
				MsgHash:    utils.ComputeMessageHash(event.Target, event.Sender, event.Value, event.Fee, event.Deadline, event.Message, event.MessageNonce).String(),
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
		case common.HexToHash(bridge_abi.RELAYED_MESSAGE_EVENT_SIGNATURE):
			event := struct {
				MsgHash common.Hash
			}{}
			// MsgHash is in topics[1]
			event.MsgHash = common.HexToHash(vLog.Topics[1].String())
			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MsgHash,
				txHash:       vLog.TxHash,
				isSuccessful: true,
			})
		case common.HexToHash(bridge_abi.FAILED_RELAYED_MESSAGE_EVENT_SIGNATURE):
			event := struct {
				MsgHash common.Hash
			}{}
			// MsgHash is in topics[1]
			event.MsgHash = common.HexToHash(vLog.Topics[1].String())
			relayedMessages = append(relayedMessages, relayedMessage{
				msgHash:      event.MsgHash,
				txHash:       vLog.TxHash,
				isSuccessful: false,
			})
		default:
			log.Error("Unknown event", "topic", vLog.Topics[0], "txHash", vLog.TxHash)
		}
	}

	return l2Messages, relayedMessages, nil
}
