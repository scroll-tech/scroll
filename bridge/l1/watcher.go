package l1

import (
	"context"
	"math/big"
	"time"

	geth "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/common/viper"

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
	vp     *viper.Viper

	// The number of new blocks to wait for a block to be confirmed
	messengerAddress common.Address
	messengerABI     *abi.ABI

	rollupAddress common.Address
	rollupABI     *abi.ABI

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight int64

	stop chan bool
}

// NewWatcher returns a new instance of Watcher. The instance will be not fully prepared,
// and still needs to be finalized and ran by calling `watcher.Start`.
func NewWatcher(ctx context.Context, client *ethclient.Client, vp *viper.Viper, db database.OrmFactory) *Watcher {
	savedHeight, err := db.GetLayer1LatestWatchedHeight()
	if err != nil {
		log.Warn("Failed to fetch height from db", "err", err)
		savedHeight = 0
	}
	startHeight := vp.GetInt64("start_height")
	if savedHeight < startHeight {
		savedHeight = startHeight
	}

	stop := make(chan bool)

	return &Watcher{
		ctx:                ctx,
		client:             client,
		db:                 db,
		vp:                 vp,
		messengerAddress:   vp.GetAddress("l1_messenger_address"),
		messengerABI:       bridge_abi.L1MessengerMetaABI,
		rollupAddress:      vp.GetAddress("rollup_contract_address"),
		rollupABI:          bridge_abi.RollupMetaABI,
		processedMsgHeight: savedHeight,
		stop:               stop,
	}
}

// Start the Watcher module.
func (w *Watcher) Start() {
	go func() {
		// trigger by timer
		ticker := time.NewTicker(w.vp.GetDuration("watcher_time_sec"))
		defer ticker.Stop()

		for ; true; <-ticker.C {
			select {
			case <-w.stop:
				return

			default:
				blockNumber, err := w.client.BlockNumber(w.ctx)
				if err != nil {
					log.Error("Failed to get block number", "err", err)
					continue
				}
				if err := w.FetchContractEvent(blockNumber); err != nil {
					log.Error("Failed to fetch bridge contract", "err", err)
				}
			}
		}
	}()
}

// Stop the Watcher module, for a graceful shutdown.
func (w *Watcher) Stop() {
	w.stop <- true
}

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (w *Watcher) FetchContractEvent(blockHeight uint64) error {
	defer func() {
		log.Info("l1 watcher fetchContractEvent", "w.processedMsgHeight", w.processedMsgHeight)
	}()

	fromBlock := w.processedMsgHeight + 1
	toBlock := int64(blockHeight) - w.vp.GetInt64("confirmations")

	contractEventsBlocksFetchLimit := w.vp.GetInt64("contract_events_blocks_fetch_limit")
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
				w.rollupAddress,
			},
			Topics: make([][]common.Hash, 1),
		}
		query.Topics[0] = make([]common.Hash, 5)
		query.Topics[0][0] = common.HexToHash(bridge_abi.SENT_MESSAGE_EVENT_SIGNATURE)
		query.Topics[0][1] = common.HexToHash(bridge_abi.RELAYED_MESSAGE_EVENT_SIGNATURE)
		query.Topics[0][2] = common.HexToHash(bridge_abi.FAILED_RELAYED_MESSAGE_EVENT_SIGNATURE)
		query.Topics[0][3] = common.HexToHash(bridge_abi.COMMIT_BATCH_EVENT_SIGNATURE)
		query.Topics[0][4] = common.HexToHash(bridge_abi.FINALIZED_BATCH_EVENT_SIGNATURE)

		logs, err := w.client.FilterLogs(w.ctx, query)
		if err != nil {
			log.Warn("Failed to get event logs", "err", err)
			return err
		}
		if len(logs) == 0 {
			w.processedMsgHeight = to
			continue
		}
		log.Info("Received new L1 events", "fromBlock", from, "toBlock", to, "cnt", len(logs))

		sentMessageEvents, relayedMessageEvents, rollupEvents, err := w.parseBridgeEventLogs(logs)
		if err != nil {
			log.Error("Failed to parse emitted events log", "err", err)
			return err
		}
		log.Info("L1 events types", "SentMessageCount", len(sentMessageEvents), "RelayedMessageCount", len(relayedMessageEvents), "RollupEventCount", len(rollupEvents))

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
			log.Error("RollupStatus.Length mismatch with BatchIDs.Length", "RollupStatus.Length", len(statuses), "BatchIDs.Length", len(batchIDs))
			return nil
		}

		for index, event := range rollupEvents {
			batchID := event.batchID.String()
			status := statuses[index]
			// only update when db status is before event status
			if event.status > status {
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

		if err = w.db.SaveL1Messages(w.ctx, sentMessageEvents); err != nil {
			return err
		}

		w.processedMsgHeight = to
	}

	return nil
}

func (w *Watcher) parseBridgeEventLogs(logs []types.Log) ([]*orm.L1Message, []relayedMessage, []rollupEvent, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l1Messages []*orm.L1Message
	var relayedMessages []relayedMessage
	var rollupEvents []rollupEvent
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
				log.Warn("Failed to unpack layer1 SentMessage event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}
			// target is in topics[1]
			event.Target = common.HexToAddress(vLog.Topics[1].String())
			l1Messages = append(l1Messages, &orm.L1Message{
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
				Layer1Hash: vLog.TxHash.Hex(),
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
		case common.HexToHash(bridge_abi.COMMIT_BATCH_EVENT_SIGNATURE):
			event := struct {
				BatchID    common.Hash
				BatchHash  common.Hash
				BatchIndex *big.Int
				ParentHash common.Hash
			}{}
			// BatchID is in topics[1]
			event.BatchID = common.HexToHash(vLog.Topics[1].String())
			err := w.rollupABI.UnpackIntoInterface(&event, "CommitBatch", vLog.Data)
			if err != nil {
				log.Warn("Failed to unpack layer1 CommitBatch event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			rollupEvents = append(rollupEvents, rollupEvent{
				batchID: event.BatchID,
				txHash:  vLog.TxHash,
				status:  orm.RollupCommitted,
			})
		case common.HexToHash(bridge_abi.FINALIZED_BATCH_EVENT_SIGNATURE):
			event := struct {
				BatchID    common.Hash
				BatchHash  common.Hash
				BatchIndex *big.Int
				ParentHash common.Hash
			}{}
			// BatchID is in topics[1]
			event.BatchID = common.HexToHash(vLog.Topics[1].String())
			err := w.rollupABI.UnpackIntoInterface(&event, "FinalizeBatch", vLog.Data)
			if err != nil {
				log.Warn("Failed to unpack layer1 FinalizeBatch event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			rollupEvents = append(rollupEvents, rollupEvent{
				batchID: event.BatchID,
				txHash:  vLog.TxHash,
				status:  orm.RollupFinalized,
			})
		default:
			log.Error("Unknown event", "topic", vLog.Topics[0], "txHash", vLog.TxHash)
		}
	}

	return l1Messages, relayedMessages, rollupEvents, nil
}
