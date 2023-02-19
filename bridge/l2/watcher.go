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
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rpc"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/utils"

	"scroll-tech/common/types"

	"scroll-tech/database"

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

	confirmations    rpc.BlockNumber
	messengerAddress common.Address
	messengerABI     *abi.ABI

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64

	stopped uint64
	stopCh  chan struct{}

	batchProposer *batchProposer
}

// NewL2WatcherClient take a l2geth instance to generate a l2watcherclient instance
func NewL2WatcherClient(ctx context.Context, client *ethclient.Client, confirmations rpc.BlockNumber, bpCfg *config.BatchProposerConfig, messengerAddress common.Address, relayer *Layer2Relayer, orm database.OrmFactory) *WatcherClient {
	savedHeight, err := orm.GetLayer2LatestWatchedHeight()
	if err != nil {
		log.Warn("fetch height from db failed", "err", err)
		savedHeight = 0
	}

	w := WatcherClient{
		ctx:                ctx,
		Client:             client,
		orm:                orm,
		processedMsgHeight: uint64(savedHeight),
		confirmations:      confirmations,
		messengerAddress:   messengerAddress,
		messengerABI:       bridge_abi.L2MessengerMetaABI,
		stopCh:             make(chan struct{}),
		stopped:            0,
		batchProposer:      newBatchProposer(bpCfg, relayer, orm),
	}

	// Initialize genesis before we do anything else
	if err := w.initializeGenesis(); err != nil {
		panic(fmt.Sprintf("failed to initialize L2 genesis batch, err: %v", err))
	}

	return &w
}

func (w *WatcherClient) initializeGenesis() error {
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

	// EIP1559 is disabled so the RPC won't return baseFeePerGas. However, l2geth
	// still uses BaseFee when calculating the block hash. If we keep it as <nil>
	// here the genesis hash will not match.
	genesis.BaseFee = big.NewInt(0)

	log.Info("retrieved L2 genesis header", "hash", genesis.Hash().String())

	blockTrace := &geth_types.BlockTrace{
		Coinbase:         nil,
		Header:           genesis,
		Transactions:     []*geth_types.TransactionData{},
		StorageTrace:     nil,
		ExecutionResults: []*geth_types.ExecutionResult{},
		MPTWitness:       nil,
	}

	batchData := types.NewGenesisBatchData([]*geth_types.BlockTrace{blockTrace})

	if err = w.batchProposer.addBatchInfoToDB(batchData); err != nil {
		log.Error("failed to add batch info to DB", "BatchHash", batchData.Hash(), "error", err)
		return err
	}

	batchHash := batchData.Hash().Hex()
	if err != nil {
		return fmt.Errorf("failed to create batch data: %v", err)
	}

	err = w.orm.UpdateProvingStatus(batchHash, types.ProvingTaskProved)
	if err != nil {
		return fmt.Errorf("failed to update genesis batch proving status: %v", err)
	}

	err = w.orm.UpdateRollupStatus(w.ctx, batchHash, types.RollupFinalized)
	if err != nil {
		return fmt.Errorf("failed to update genesis batch rollup status: %v", err)
	}

	log.Info("successfully imported genesis batch")

	return nil
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
					number, err := utils.GetLatestConfirmedBlockNumber(ctx, w.Client, w.confirmations)
					if err != nil {
						log.Error("failed to get block number", "err", err)
						continue
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
					number, err := utils.GetLatestConfirmedBlockNumber(ctx, w.Client, w.confirmations)
					if err != nil {
						log.Error("failed to get block number", "err", err)
						continue
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
	heightInDB, err := w.orm.GetBlockTracesLatestHeight()
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
	var traces []*geth_types.BlockTrace

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
				err = w.orm.UpdateLayer1StatusAndLayer2Hash(w.ctx, msg.msgHash.String(), types.MsgConfirmed, msg.txHash.String())
			} else {
				// failed
				err = w.orm.UpdateLayer1StatusAndLayer2Hash(w.ctx, msg.msgHash.String(), types.MsgFailed, msg.txHash.String())
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

func (w *WatcherClient) parseBridgeEventLogs(logs []geth_types.Log) ([]*types.L2Message, []relayedMessage, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l2Messages []*types.L2Message
	var relayedMessages []relayedMessage
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case common.HexToHash(bridge_abi.SentMessageEventSignature):
			event := struct {
				Sender       common.Address
				Target       common.Address
				Value        *big.Int // uint256
				MessageNonce *big.Int // uint256
				Message      []byte
			}{}

			err := w.messengerABI.UnpackIntoInterface(&event, "SentMessage", vLog.Data)
			if err != nil {
				log.Error("failed to unpack layer2 SentMessage event", "err", err)
				return l2Messages, relayedMessages, err
			}
			// target is in topics[1]
			event.Target = common.HexToAddress(vLog.Topics[1].String())
			l2Messages = append(l2Messages, &types.L2Message{
				Nonce: event.MessageNonce.Uint64(),
				// MsgHash:    utils.ComputeMessageHash(event.Sender, event.Target, event.Value, event.Fee, event.Deadline, event.Message, event.MessageNonce).String(),
				// MsgHash: // todo: use encodeXDomainData from contracts,
				Height:     vLog.BlockNumber,
				Sender:     event.Sender.String(),
				Value:      event.Value.String(),
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
