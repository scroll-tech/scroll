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

	"scroll-tech/database"
	"scroll-tech/database/orm"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/utils"
)

const (
	// SENT_MESSAGE_EVENT_SIGNATURE = keccak256("SentMessage(address,address,uint256,uint256,uint256,bytes,uint256,uint256)")
	SENT_MESSAGE_EVENT_SIGNATURE = "806b28931bc6fbe6c146babfb83d5c2b47e971edb43b4566f010577a0ee7d9f4"

	// RELAYED_MESSAGE_EVENT_SIGNATURE = keccak256("RelayedMessage(bytes32)")
	RELAYED_MESSAGE_EVENT_SIGNATURE = "4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c"

	// FAILED_RELAYED_MESSAGE_EVENT_SIGNATURE = keccak256("FailedRelayedMessage(bytes32)")
	FAILED_RELAYED_MESSAGE_EVENT_SIGNATURE = "99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f"

	// COMMIT_BATCH_EVENT_SIGNATURE = keccak256("CommitBatch(bytes32,bytes32,uint256,bytes32)")
	COMMIT_BATCH_EVENT_SIGNATURE = "a26d4bd91c4c2eff3b1bf542129607d782506fc1950acfab1472a20d28c06596"

	// FINALIZED_BATCH_EVENT_SIGNATURE = keccak256("FinalizeBatch(bytes32,bytes32,uint256,bytes32)")
	FINALIZED_BATCH_EVENT_SIGNATURE = "e20f311a96205960de4d2bb351f7729e5136fa36ae64d7f736c67ddc4ca4cd4b"
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
func NewWatcher(ctx context.Context, client *ethclient.Client, startHeight uint64, confirmations uint64, messengerAddress common.Address, rollupAddress common.Address, db database.OrmFactory) *Watcher {
	savedHeight, err := db.GetLayer1LatestWatchedHeight()
	if err != nil {
		log.Warn("Failed to fetch height from db", "err", err)
		savedHeight = 0
	}
	if savedHeight < int64(startHeight) {
		savedHeight = int64(startHeight)
	}

	stop := make(chan bool)

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
	query.Topics[0][0] = common.HexToHash(SENT_MESSAGE_EVENT_SIGNATURE)
	query.Topics[0][1] = common.HexToHash(RELAYED_MESSAGE_EVENT_SIGNATURE)
	query.Topics[0][2] = common.HexToHash(FAILED_RELAYED_MESSAGE_EVENT_SIGNATURE)
	query.Topics[0][3] = common.HexToHash(COMMIT_BATCH_EVENT_SIGNATURE)
	query.Topics[0][4] = common.HexToHash(FINALIZED_BATCH_EVENT_SIGNATURE)

	logs, err := w.client.FilterLogs(w.ctx, query)
	if err != nil {
		log.Warn("Failed to get event logs", "err", err)
		return err
	}
	if len(logs) == 0 {
		r.processedMsgHeight = uint64(toBlock)
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

	// Update relayed message first to make sure we don't forget to update submited message.
	// Since, we always start sync from the latest unprocessed message.
	for _, msg := range relayedMessageEvents {
		if msg.isSuccessful {
			// succeed
			err = w.db.UpdateLayer1StatusAndLayer2Hash(w.ctx, msg.msgHash.String(), msg.txHash.String(), orm.MsgConfirmed)
		} else {
			// failed
			err = w.db.UpdateLayer1StatusAndLayer2Hash(w.ctx, msg.msgHash.String(), msg.txHash.String(), orm.MsgFailed)
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

	var parsedlogs []*orm.L1Message
	var relayedMessages []relayedMessage
	var rollupEvents []rollupEvent
	for _, vLog := range logs {
		if vLog.Topics[0] == common.HexToHash(SENT_MESSAGE_EVENT_SIGNATURE) {
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
				return parsedlogs, relayedMessages, rollupEvents, err
			}
			// target is in topics[1]
			event.Target = common.HexToAddress(vLog.Topics[1].String())
			parsedlogs = append(parsedlogs, &orm.L1Message{
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
		} else if vLog.Topics[0] == common.HexToHash(RELAYED_MESSAGE_EVENT_SIGNATURE) {
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
		} else if vLog.Topics[0] == common.HexToHash(FAILED_RELAYED_MESSAGE_EVENT_SIGNATURE) {
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
		} else if vLog.Topics[0] == common.HexToHash(COMMIT_BATCH_EVENT_SIGNATURE) {
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
				return parsedlogs, relayedMessages, rollupEvents, err
			}

			rollupEvents = append(rollupEvents, rollupEvent{
				batchID: event.BatchID,
				txHash:  vLog.TxHash,
				status:  orm.RollupCommitted,
			})
		} else if vLog.Topics[0] == common.HexToHash(FINALIZED_BATCH_EVENT_SIGNATURE) {
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
				return parsedlogs, relayedMessages, rollupEvents, err
			}

			rollupEvents = append(rollupEvents, rollupEvent{
				batchID: event.BatchID,
				txHash:  vLog.TxHash,
				status:  orm.RollupFinalized,
			})
		}
	}

	return parsedlogs, relayedMessages, rollupEvents, nil
}
