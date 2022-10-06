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

	"scroll-tech/store/orm"
)

const (
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10

	// keccak256("SentMessage(address,address,uint256,uint256,uint256,bytes,uint256,uint256)")
	sentMessageEventSignature = "806b28931bc6fbe6c146babfb83d5c2b47e971edb43b4566f010577a0ee7d9f4"
)

// Watcher will listen for smart contract events from Eth L1.
type Watcher struct {
	ctx    context.Context
	client *ethclient.Client
	db     orm.Layer1MessageOrm

	// The number of new blocks to wait for a block to be confirmed
	confirmations    uint64
	messengerAddress common.Address
	messengerABI     *abi.ABI

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64

	stop chan bool
}

// NewWatcher returns a new instance of Watcher. The instance will be not fully prepared,
// and still needs to be finalized and ran by calling `watcher.Start`.
func NewWatcher(ctx context.Context, client *ethclient.Client, startHeight uint64, confirmations uint64, messengerAddress common.Address, messengerABI *abi.ABI, db orm.Layer1MessageOrm) *Watcher {
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
		messengerABI:       messengerABI,
		processedMsgHeight: uint64(savedHeight),
		stop:               stop,
	}
}

// Start the Watcher module.
func (r *Watcher) Start() {
	go func() {
		// trigger by timer
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				blockNumber, err := r.client.BlockNumber(r.ctx)
				if err != nil {
					log.Error("Failed to get block number", "err", err)
				}
				if err := r.fetchContractEvent(blockNumber); err != nil {
					log.Error("Failed to fetch bridge contract", "err", err)
				}
			case <-r.stop:
				return
			}
		}
	}()
}

// Stop the Watcher module, for a graceful shutdown.
func (r *Watcher) Stop() {
	r.stop <- true
}

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (r *Watcher) fetchContractEvent(blockHeight uint64) error {
	fromBlock := int64(r.processedMsgHeight) + 1
	toBlock := int64(blockHeight) - int64(r.confirmations)

	if toBlock < fromBlock {
		return nil
	}

	// warning: uint int conversion...
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlock), // inclusive
		ToBlock:   big.NewInt(toBlock),   // inclusive
		Addresses: []common.Address{
			r.messengerAddress,
		},
		Topics: make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 1)
	query.Topics[0][0] = common.HexToHash(sentMessageEventSignature)

	logs, err := r.client.FilterLogs(r.ctx, query)
	if err != nil {
		log.Warn("Failed to get event logs", "err", err)
		return err
	}
	if len(logs) == 0 {
		return nil
	}
	log.Info("Received new L1 messages", "fromBlock", fromBlock, "toBlock", toBlock,
		"cnt", len(logs))

	eventLogs, err := parseBridgeEventLogs(logs, r.messengerABI)
	if err != nil {
		log.Error("Failed to parse emitted events log", "err", err)
		return err
	}

	err = r.db.SaveLayer1Messages(r.ctx, eventLogs)
	if err == nil {
		r.processedMsgHeight = uint64(toBlock)
	}
	return err
}

func parseBridgeEventLogs(logs []types.Log, messengerABI *abi.ABI) ([]*orm.Layer1Message, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var parsedlogs []*orm.Layer1Message
	for _, vLog := range logs {
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

		err := messengerABI.UnpackIntoInterface(&event, "SentMessage", vLog.Data)
		if err != nil {
			log.Warn("Failed to unpack layer1 SentMessage event", "err", err)
			return parsedlogs, err
		}
		// target is in topics[1]
		event.Target = common.HexToAddress(vLog.Topics[1].String())
		parsedlogs = append(parsedlogs, &orm.Layer1Message{
			Nonce:      event.MessageNonce.Uint64(),
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
	}

	return parsedlogs, nil
}
