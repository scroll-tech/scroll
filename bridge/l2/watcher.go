package l2

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/jmoiron/sqlx"
	geth "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"

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
func NewL2WatcherClient(ctx context.Context, client *ethclient.Client, confirmations uint64, bpCfg *config.BatchProposerConfig, messengerAddress, messageQueueAddress, blockContainerAddress common.Address, orm database.OrmFactory) (*WatcherClient, error) {
	savedHeight, err := orm.GetLayer2LatestWatchedHeight()
	if err != nil {
		log.Warn("fetch height from db failed", "err", err)
		savedHeight = 0
	}

	withdrawTrie := NewWithdrawTrie()
	currentMessageNonce, err := orm.GetLayer2LatestMessageNonce()
	if err != nil {
		log.Warn("fetch message nonce from db failed", "err", err)
		return nil, err
	}

	if currentMessageNonce != -1 {
		msg, err := orm.GetL2MessageByNonce(uint64(currentMessageNonce))
		if err != nil {
			log.Warn("fetch message by nonce from db failed", "err", err)
			return nil, err
		}
		// fetch and rebuild from message db
		proofBytes, err := orm.GetMessageProofByNonce(uint64(currentMessageNonce))
		if err != nil {
			log.Warn("fetch message proof from db failed", "err", err)
			return nil, err
		}
		if len(proofBytes)%32 != 0 {
			log.Warn("proof string has wrong length", "length", len(proofBytes))
			return nil, errors.New("proof string with wrong length")
		}
		withdrawTrie.Initialize(uint64(currentMessageNonce), common.HexToHash(msg.MsgHash), proofBytes)
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

		withdrawTrie: withdrawTrie,

		stopCh:        make(chan struct{}),
		stopped:       0,
		batchProposer: newBatchProposer(bpCfg, orm),
	}, nil
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

	var dbTx *sqlx.Tx
	var dbTxErr error
	defer func() {
		if dbTxErr != nil {
			if err := dbTx.Rollback(); err != nil {
				log.Error("dbTx.Rollback()", "err", err)
			}
		}
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
				w.messageQueueAddress,
				w.blockContainerAddress,
			},
			Topics: make([][]common.Hash, 1),
		}
		query.Topics[0] = make([]common.Hash, 5)
		query.Topics[0][0] = bridge_abi.L2SentMessageEventSignature
		query.Topics[0][1] = bridge_abi.L2RelayedMessageEventSignature
		query.Topics[0][2] = bridge_abi.L2FailedRelayedMessageEventSignature
		query.Topics[0][3] = bridge_abi.L2AppendMessageEventSignature
		query.Topics[0][4] = bridge_abi.L2ImportBlockEventSignature

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
		log.Info("received new L2 messages", "fromBlock", from, "toBlock", to,
			"cnt", len(logs))

		sentMessageEvents, relayedMessageEvents, importedBlockEvents, err := w.parseBridgeEventLogs(logs)
		if err != nil {
			log.Error("failed to parse emitted event log", "err", err)
			return
		}

		// Update imported block first to make sure we don't forget to update importing blocks.
		for _, block := range importedBlockEvents {
			err = w.orm.UpdateL1BlockStatusAndImportTxHash(w.ctx, block.blockHash.String(), orm.L1BlockImported, block.txHash.String())
			if err != nil {
				log.Error("Failed to update l1 block status and import tx hash", "err", err)
				return
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
				return
			}
		}

		dbTx, err = w.orm.Beginx()
		if err != nil {
			return
		}

		// group sentMessageEvents by block height
		index := 0
		nonce := w.withdrawTrie.NextMessageNonce
		for height := from; height <= to; height++ {
			var hashes []common.Hash
			var msgs []*orm.L2Message
			for ; index < len(sentMessageEvents) && sentMessageEvents[index].Height == uint64(height); index++ {
				if nonce != sentMessageEvents[index].Nonce {
					log.Error("nonce mismatch", "expected", nonce, "found", sentMessageEvents[index].Nonce)
					return
				}
				hashes = append(hashes, common.HexToHash(sentMessageEvents[index].MsgHash))
				msgs = append(msgs, sentMessageEvents[index])
				nonce++
			}
			proofBytes := w.withdrawTrie.AppendMessages(hashes)
			for i := 0; i < len(hashes); i++ {
				msgs[i].Proof = common.Bytes2Hex(proofBytes[i])
			}

			// save message root in block
			dbTxErr = w.orm.SetMessageRootForBlocksInDBTx(dbTx, []uint64{uint64(height)}, w.withdrawTrie.MessageRoot().String())
			if dbTxErr != nil {
				log.Error("SetBatchIDForBlocksInDBTx failed", "error", dbTxErr)
				return
			}

			// save l2 messages
			dbTxErr = w.orm.SaveL2MessagesInDbTx(w.ctx, dbTx, msgs)
			if dbTxErr != nil {
				log.Error("SaveL2MessagesInDbTx failed", "error", dbTxErr)
				return
			}
		}

		dbTxErr = dbTx.Commit()
		if dbTxErr != nil {
			log.Error("dbTx.Commit failed", "error", dbTxErr)
			return
		}

		w.processedMsgHeight = uint64(to)
		bridgeL2MsgSyncHeightGauge.Update(to)
	}
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
		case bridge_abi.L2SentMessageEventSignature:
			event := bridge_abi.L2SentMessageEvent{}
			err := utils.UnpackLog(w.messengerABI, &event, "SentMessage", vLog)
			if err != nil {
				log.Error("failed to unpack layer2 SentMessage event", "err", err)
				return l2Messages, relayedMessages, importedBlocks, err
			}
			computedMsgHash := utils.ComputeMessageHash(
				event.Sender,
				event.Target,
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
			event := bridge_abi.L2RelayedMessageEvent{}
			err := utils.UnpackLog(w.messengerABI, &event, "RelayedMessage", vLog)
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
			event := bridge_abi.L2FailedRelayedMessageEvent{}
			err := utils.UnpackLog(w.messengerABI, &event, "FailedRelayedMessage", vLog)
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
			event := bridge_abi.L2ImportBlockEvent{}
			err := utils.UnpackLog(w.blockContainerABI, &event, "ImportBlock", vLog)
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
			event := bridge_abi.L2AppendMessageEvent{}
			err := utils.UnpackLog(w.messageQueueABI, &event, "AppendMessage", vLog)
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
