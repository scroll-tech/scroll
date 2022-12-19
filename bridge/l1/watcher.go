package l1

import (
	"context"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/spf13/viper"

	"scroll-tech/database"
	"scroll-tech/database/orm"

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

	// The number of new blocks to wait for a block to be confirmed
	confirmations    uint64
	messengerAddress common.Address
	messengerABI     *abi.ABI

	rollupAddress common.Address
	rollupABI     *abi.ABI

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64

	stop chan bool
}

// NewWatcher returns a new instance of Watcher. The instance will be not fully prepared,
// and still needs to be finalized and ran by calling `watcher.Start`.
func NewWatcher(ctx context.Context, client *ethclient.Client, db database.OrmFactory) *Watcher {
	savedHeight, err := db.GetLayer1LatestWatchedHeight()
	if err != nil {
		log.Warn("Failed to fetch height from db", "err", err)
		savedHeight = 0
	}
	startHeight := viper.GetInt64("l1_config.start_height")
	if savedHeight < startHeight {
		savedHeight = startHeight
	}

	stop := make(chan bool)
	confirmations := uint64(viper.GetInt64("l1_config.confirmations"))
	messengerAddress := common.HexToAddress(viper.GetString("l1_config.l1_messenger_address"))
	rollupAddress := common.HexToAddress(viper.GetString("l1_config.relayer_config.rollup_contract_address"))

	return &Watcher{
		ctx:                ctx,
		client:             client,
		db:                 db,
		confirmations:      confirmations,
		messengerAddress:   messengerAddress,
		messengerABI:       bridge_abi.L1MessengerMetaABI,
		rollupAddress:      rollupAddress,
		rollupABI:          bridge_abi.RollupMetaABI,
		processedMsgHeight: uint64(savedHeight),
		stop:               stop,
	}
}

// Start the Watcher module.
func (w *Watcher) Start() {
	go func() {
		// trigger by timer
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				blockNumber, err := w.client.BlockNumber(w.ctx)
				if err != nil {
					log.Error("Failed to get block number", "err", err)
				}
				if err := w.fetchContractEvent(blockNumber); err != nil {
					log.Error("Failed to fetch bridge contract", "err", err)
				}
			case <-w.stop:
				return
			}
		}
	}()
}

// Stop the Watcher module, for a graceful shutdown.
func (w *Watcher) Stop() {
	w.stop <- true
}

const contractEventsBlocksFetchLimit = int64(10)

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (w *Watcher) fetchContractEvent(blockHeight uint64) error {
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
		w.processedMsgHeight = uint64(toBlock)
		return nil
	}
	log.Info("Received new L1 messages", "fromBlock", fromBlock, "toBlock", toBlock,
		"cnt", len(logs))

	sentMessageEvents, relayedMessageEvents, rollupEvents, err := w.parseBridgeEventLogs(logs)
	if err != nil {
		log.Error("Failed to parse emitted events log", "err", err)
		return err
	}

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
		log.Error("RollupStatus.Length mismatch with BatchIDs.Length")
		return nil
	}

	for index, event := range rollupEvents {
		batchID := event.batchID.String()
		status := statuses[index]
		if event.status != status {
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

	err = w.db.SaveL1Messages(w.ctx, sentMessageEvents)
	if err == nil {
		w.processedMsgHeight = uint64(toBlock)
	}
	return err
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
