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
	"github.com/scroll-tech/go-ethereum/metrics"

	"scroll-tech/common/bigint"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/utils"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/bridge/config"
)

// Metrics
var (
	bridgeL2MsgSyncHeightGauge = metrics.NewRegisteredGauge("bridge/l2/msg/sync/height", nil)
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

	confirmations    uint64
	messengerAddress common.Address
	messengerABI     *abi.ABI

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64

	stopped uint64
	stopCh  chan struct{}

	batchProposer *batchProposer
}

// NewL2WatcherClient take a l2geth instance to generate a l2watcherclient instance
func NewL2WatcherClient(ctx context.Context, client *ethclient.Client, confirmations uint64, bpCfg *config.BatchProposerConfig, messengerAddress common.Address, orm database.OrmFactory) *WatcherClient {
	savedHeight, err := orm.GetLayer2LatestWatchedHeight()
	if err != nil {
		log.Warn("fetch height from db failed", "err", err)
		savedHeight = big.NewInt(0)
	}

	return &WatcherClient{
		ctx:                ctx,
		Client:             client,
		orm:                orm,
		processedMsgHeight: savedHeight.Uint64(),
		confirmations:      confirmations,
		messengerAddress:   messengerAddress,
		messengerABI:       bridge_abi.L2MessengerMetaABI,
		stopCh:             make(chan struct{}),
		stopped:            0,
		batchProposer:      newBatchProposer(bpCfg, orm),
	}
}

// Start the Listening process
func (w *WatcherClient) Start() {
	go func() {
		if reflect.ValueOf(w.orm).IsNil() {
			panic("must run L2 watcher with DB")
		}

		ctx, cancel := context.WithCancel(w.ctx)

		// trace fetcher loop
		go func(ctx context.Context) {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return

				case <-ticker.C:
					// get current height
					number, err := w.BlockNumber(ctx)
					if err != nil {
						log.Error("failed to get_BlockNumber", "err", err)
						continue
					}

					if number >= w.confirmations {
						number = number - w.confirmations
					} else {
						number = 0
					}

					w.tryFetchRunningMissingBlocks(ctx, number)
				}
			}
		}(ctx)

		// event fetcher loop
		go func(ctx context.Context) {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return

				case <-ticker.C:
					// get current height
					number, err := w.BlockNumber(ctx)
					if err != nil {
						log.Error("failed to get_BlockNumber", "err", err)
						continue
					}

					if number >= w.confirmations {
						number = number - w.confirmations
					} else {
						number = 0
					}

					w.FetchContractEvent(number)
				}
			}
		}(ctx)

		// batch proposer loop
		go func(ctx context.Context) {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return

				case <-ticker.C:
					w.batchProposer.tryProposeBatch()
				}
			}
		}(ctx)

		<-w.stopCh
		cancel()
	}()
}

// Stop the Watcher module, for a graceful shutdown.
func (w *WatcherClient) Stop() {
	w.stopCh <- struct{}{}
}

const blockTracesFetchLimit = uint64(10)

// try fetch missing blocks if inconsistent
func (w *WatcherClient) tryFetchRunningMissingBlocks(ctx context.Context, blockHeight uint64) {
	// Get newest block in DB. must have blocks at that time.
	// Don't use "block_trace" table "trace" column's BlockTrace.Number,
	// because it might be empty if the corresponding rollup_result is finalized/finalization_skipped
	heightInDBBig, err := w.orm.GetBlockTracesLatestHeight()
	heightInDB := heightInDBBig.Int64()
	if err != nil {
		log.Error("failed to GetBlockTracesLatestHeight", "err", err)
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
	}
}

func (w *WatcherClient) getAndStoreBlockTraces(ctx context.Context, from, to uint64) error {
	var traces []*types.BlockTrace

	for number := from; number <= to; number++ {
		log.Debug("retrieving block trace", "height", number)
		trace, err2 := w.GetBlockTraceByNumber(ctx, big.NewInt(int64(number)))
		if err2 != nil {
			return fmt.Errorf("failed to GetBlockResultByHash: %v. number: %v", err2, number)
		}
		log.Info("retrieved block trace", "height", trace.Header.Number, "hash", trace.Header.Hash().String())

		traces = append(traces, trace)

	}
	if len(traces) > 0 {
		if err := w.orm.InsertBlockTraces(traces); err != nil {
			return fmt.Errorf("failed to batch insert BlockTraces: %v", err)
		}
	}

	return nil
}

const contractEventsBlocksFetchLimit = int64(10)

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (w *WatcherClient) FetchContractEvent(blockHeight uint64) {
	defer func() {
		log.Info("l2 watcher fetchContractEvent", "w.processedMsgHeight", w.processedMsgHeight)
	}()

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
			},
			Topics: make([][]common.Hash, 1),
		}
		query.Topics[0] = make([]common.Hash, 3)
		query.Topics[0][0] = common.HexToHash(bridge_abi.SentMessageEventSignature)
		query.Topics[0][1] = common.HexToHash(bridge_abi.RelayedMessageEventSignature)
		query.Topics[0][2] = common.HexToHash(bridge_abi.FailedRelayedMessageEventSignature)

		logs, err := w.FilterLogs(w.ctx, query)
		if err != nil {
			log.Error("failed to get event logs", "err", err)
			return
		}
		if len(logs) == 0 {
			w.processedMsgHeight = uint64(to)
			bridgeL2MsgSyncHeightGauge.Update(to)
			continue
		}
		log.Info("received new L2 messages", "fromBlock", from, "toBlock", to, "cnt", len(logs))

		sentMessageEvents, relayedMessageEvents, err := w.parseBridgeEventLogs(logs)
		if err != nil {
			log.Error("failed to parse emitted event log", "err", err)
			return
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
				return
			}
		}

		if err = w.orm.SaveL2Messages(w.ctx, sentMessageEvents); err != nil {
			log.Error("failed to save l2 messages", "err", err)
			return
		}

		w.processedMsgHeight = uint64(to)
		bridgeL2MsgSyncHeightGauge.Update(to)
	}
}

func (w *WatcherClient) parseBridgeEventLogs(logs []types.Log) ([]*orm.L2Message, []relayedMessage, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l2Messages []*orm.L2Message
	var relayedMessages []relayedMessage
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case common.HexToHash(bridge_abi.SentMessageEventSignature):
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
				MsgHash:    utils.ComputeMessageHash(event.Sender, event.Target, event.Value, event.Fee, event.Deadline, event.Message, event.MessageNonce).String(),
				Height:     bigint.NewUInt(vLog.BlockNumber),
				Sender:     event.Sender.String(),
				Value:      event.Value.String(),
				Fee:        event.Fee.String(),
				GasLimit:   event.GasLimit.Uint64(),
				Deadline:   event.Deadline.Uint64(),
				Target:     event.Target.String(),
				Calldata:   common.Bytes2Hex(event.Message),
				Layer2Hash: vLog.TxHash.Hex(),
			})
		case common.HexToHash(bridge_abi.RelayedMessageEventSignature):
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
		case common.HexToHash(bridge_abi.FailedRelayedMessageEventSignature):
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
