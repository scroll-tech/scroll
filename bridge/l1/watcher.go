package l1

import (
	"context"
	"math/big"
	"time"

	geth "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/types"

	"scroll-tech/database"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/utils"
)

var (
	bridgeL1MsgSyncHeightGauge = metrics.NewRegisteredGauge("bridge/l1/msg/sync/height", nil)
)

type relayedMessage struct {
	msgHash      common.Hash
	txHash       common.Hash
	isSuccessful bool
}

type rollupEvent struct {
	batchID common.Hash
	txHash  common.Hash
	status  types.RollupStatus
}

// Watcher will listen for smart contract events from Eth L1.
type Watcher struct {
	ctx    context.Context
	client *ethclient.Client
	db     database.OrmFactory

	// The number of new blocks to wait for a block to be confirmed
	confirmations    rpc.BlockNumber
	messengerAddress common.Address
	messengerABI     *abi.ABI

	scrollchainAddress common.Address
	scrollchainABI     *abi.ABI

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64
	// The height of the block that the watcher has retrieved header rlp
	processedBlockHeight uint64

	stop chan bool
}

// NewWatcher returns a new instance of Watcher. The instance will be not fully prepared,
// and still needs to be finalized and ran by calling `watcher.Start`.
func NewWatcher(ctx context.Context, client *ethclient.Client, startHeight uint64, confirmations rpc.BlockNumber, messengerAddress common.Address, scrollchainAddress common.Address, db database.OrmFactory) *Watcher {
	savedHeight, err := db.GetLayer1LatestWatchedHeight()
	if err != nil {
		log.Warn("Failed to fetch height from db", "err", err)
		savedHeight = 0
	}
	if savedHeight < int64(startHeight) {
		savedHeight = int64(startHeight)
	}

	savedL1BlockHeight, err := db.GetLatestL1BlockHeight()
	if err != nil {
		log.Warn("Failed to fetch latest L1 block height from db", "err", err)
		savedL1BlockHeight = 0
	}
	if savedL1BlockHeight < startHeight {
		savedL1BlockHeight = startHeight
	}

	stop := make(chan bool)

	return &Watcher{
		ctx:                  ctx,
		client:               client,
		db:                   db,
		confirmations:        confirmations,
		messengerAddress:     messengerAddress,
		messengerABI:         bridge_abi.L1ScrollMessengerABI,
		scrollchainAddress:   scrollchainAddress,
		scrollchainABI:       bridge_abi.ScrollchainABI,
		processedMsgHeight:   uint64(savedHeight),
		processedBlockHeight: savedL1BlockHeight,
		stop:                 stop,
	}
}

// Start the Watcher module.
func (w *Watcher) Start() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for ; true; <-ticker.C {
			select {
			case <-w.stop:
				return

			default:
				number, err := utils.GetLatestConfirmedBlockNumber(w.ctx, w.client, w.confirmations)
				if err != nil {
					log.Error("failed to get block number", "err", err)
					continue
				}

				if err := w.fetchBlockHeader(number); err != nil {
					log.Error("Failed to fetch L1 block header", "lastest", number, "err", err)
				}

				if err := w.FetchContractEvent(number); err != nil {
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

const contractEventsBlocksFetchLimit = int64(10)

// fetchBlockHeader pull latest L1 blocks and save in DB
func (w *Watcher) fetchBlockHeader(blockHeight uint64) error {
	fromBlock := int64(w.processedBlockHeight) + 1
	toBlock := int64(blockHeight) - int64(w.confirmations)
	if toBlock < fromBlock {
		return nil
	}
	if toBlock > fromBlock+contractEventsBlocksFetchLimit {
		toBlock = fromBlock + contractEventsBlocksFetchLimit - 1
	}

	var blocks []*types.L1BlockInfo
	var err error
	height := fromBlock
	for ; height <= toBlock; height++ {
		var block *geth_types.Block
		block, err = w.client.BlockByNumber(w.ctx, big.NewInt(height))
		if err != nil {
			log.Warn("Failed to get block", "height", height, "err", err)
			break
		}
		/*
			var headerRLPBytes []byte
			headerRLPBytes, err = rlp.EncodeToBytes(block.Header())
			if err != nil {
				log.Warn("Failed to rlp encode header", "height", height, "err", err)
				break
			}
		*/
		blocks = append(blocks, &types.L1BlockInfo{
			Number: uint64(height),
			Hash:   block.Hash().String(),
			// no need to import l1 blocks now
			// HeaderRLP:       common.Bytes2Hex(headerRLPBytes),
			BaseFee: block.BaseFee().Uint64(),
		})
	}

	// failed at first block, return with the error
	if height == fromBlock {
		return err
	}
	toBlock = height - 1

	// insert succeed blocks
	err = w.db.InsertL1Blocks(w.ctx, blocks)
	if err != nil {
		log.Warn("Failed to insert L1 block to db", "fromBlock", fromBlock, "toBlock", toBlock, "err", err)
		return err
	}

	// update processed height
	w.processedBlockHeight = uint64(toBlock)
	return nil
}

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (w *Watcher) FetchContractEvent(blockHeight uint64) error {
	defer func() {
		log.Info("l1 watcher fetchContractEvent", "w.processedMsgHeight", w.processedMsgHeight)
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
				w.scrollchainAddress,
			},
			Topics: make([][]common.Hash, 1),
		}
		query.Topics[0] = make([]common.Hash, 5)
		query.Topics[0][0] = common.HexToHash(bridge_abi.QueueTransactionEventSignature)
		query.Topics[0][1] = common.HexToHash(bridge_abi.RelayedMessageEventSignature)
		query.Topics[0][2] = common.HexToHash(bridge_abi.FailedRelayedMessageEventSignature)
		query.Topics[0][3] = common.HexToHash(bridge_abi.CommitBatchEventSignature)
		query.Topics[0][4] = common.HexToHash(bridge_abi.CommitBatchesEventSignature)
		query.Topics[0][5] = common.HexToHash(bridge_abi.FinalizedBatchEventSignature)

		logs, err := w.client.FilterLogs(w.ctx, query)
		if err != nil {
			log.Warn("Failed to get event logs", "err", err)
			return err
		}
		if len(logs) == 0 {
			w.processedMsgHeight = uint64(to)
			bridgeL1MsgSyncHeightGauge.Update(to)
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
				if event.status == types.RollupFinalized {
					err = w.db.UpdateFinalizeTxHashAndRollupStatus(w.ctx, batchID, event.txHash.String(), event.status)
				} else if event.status == types.RollupCommitted {
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
				err = w.db.UpdateLayer2StatusAndLayer1Hash(w.ctx, msg.msgHash.String(), types.MsgConfirmed, msg.txHash.String())
			} else {
				// failed
				err = w.db.UpdateLayer2StatusAndLayer1Hash(w.ctx, msg.msgHash.String(), types.MsgFailed, msg.txHash.String())
			}
			if err != nil {
				log.Error("Failed to update layer1 status and layer2 hash", "err", err)
				return err
			}
		}

		if err = w.db.SaveL1Messages(w.ctx, sentMessageEvents); err != nil {
			return err
		}

		w.processedMsgHeight = uint64(to)
		bridgeL1MsgSyncHeightGauge.Update(to)
	}

	return nil
}

func (w *Watcher) parseBridgeEventLogs(logs []geth_types.Log) ([]*types.L1Message, []relayedMessage, []rollupEvent, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l1Messages []*types.L1Message
	var relayedMessages []relayedMessage
	var rollupEvents []rollupEvent
	for _, vLog := range logs {
		switch vLog.Topics[0] {

		case common.HexToHash(bridge_abi.QueueTransactionEventSignature):
			event := struct {
				Sender     common.Address
				Target     common.Address
				Value      *big.Int // uint256
				QueueIndex *big.Int // uint256
				GasLimit   *big.Int // uint256
				Data       []byte
			}{}

			err := w.messengerABI.UnpackIntoInterface(&event, "QueueTransaction", vLog.Data)
			if err != nil {
				log.Warn("Failed to unpack layer1 QueueTransaction event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}
			// target is in topics[1]
			event.Target = common.HexToAddress(vLog.Topics[1].String())
			l1Messages = append(l1Messages, &types.L1Message{
				QueueIndex: event.QueueIndex.Uint64(),
				// MsgHash:    utils.ComputeMessageHash(event.Sender, event.Target, event.Value, event.Fee, event.Deadline, event.Message, event.MessageNonce).String(),
				// MsgHash: // todo: use encodeXDomainData from contracts,
				Height:     vLog.BlockNumber,
				Sender:     event.Sender.String(),
				Value:      event.Value.String(),
				Target:     event.Target.String(),
				Calldata:   common.Bytes2Hex(event.Data),
				GasLimit:   event.GasLimit.Uint64(),
				Layer1Hash: vLog.TxHash.Hex(),
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
		case common.HexToHash(bridge_abi.CommitBatchEventSignature):
			event := struct {
				BatchID   common.Hash
				BatchHash common.Hash
			}{}
			// BatchID is in topics[1]
			event.BatchID = common.HexToHash(vLog.Topics[1].String())
			err := w.scrollchainABI.UnpackIntoInterface(&event, "CommitBatch", vLog.Data)
			if err != nil {
				log.Warn("Failed to unpack layer1 CommitBatch event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			rollupEvents = append(rollupEvents, rollupEvent{
				batchID: event.BatchID,
				txHash:  vLog.TxHash,
				status:  types.RollupCommitted,
			})
		case common.HexToHash(bridge_abi.CommitBatchesEventSignature):
			event := struct {
				BatchID   common.Hash
				BatchHash common.Hash
			}{}
			// BatchID is in topics[1]
			event.BatchID = common.HexToHash(vLog.Topics[1].String())
			err := w.scrollchainABI.UnpackIntoInterface(&event, "CommitBatches", vLog.Data)
			if err != nil {
				log.Warn("Failed to unpack layer1 CommitBatches event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			rollupEvents = append(rollupEvents, rollupEvent{
				batchID: event.BatchID,
				txHash:  vLog.TxHash,
				status:  types.RollupCommitted,
			})
		case common.HexToHash(bridge_abi.FinalizedBatchEventSignature):
			event := struct {
				BatchID   common.Hash
				BatchHash common.Hash
			}{}
			// BatchID is in topics[1]
			event.BatchID = common.HexToHash(vLog.Topics[1].String())
			err := w.scrollchainABI.UnpackIntoInterface(&event, "FinalizeBatch", vLog.Data)
			if err != nil {
				log.Warn("Failed to unpack layer1 FinalizeBatch event", "err", err)
				return l1Messages, relayedMessages, rollupEvents, err
			}

			rollupEvents = append(rollupEvents, rollupEvent{
				batchID: event.BatchID,
				txHash:  vLog.TxHash,
				status:  types.RollupFinalized,
			})
		default:
			log.Error("Unknown event", "topic", vLog.Topics[0], "txHash", vLog.TxHash)
		}
	}

	return l1Messages, relayedMessages, rollupEvents, nil
}
