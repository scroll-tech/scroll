package l2

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database"
	"scroll-tech/database/orm"
)

const (
	// BufferSize for the BlockResult channel
	BufferSize = 16

	// BlockResultCacheSize for the latest handled blockresults in memory.
	BlockResultCacheSize = 64

	// keccak256("SentMessage(address,address,uint256,uint256,uint256,bytes,uint256,uint256)")
	sentMessageEventSignature = "806b28931bc6fbe6c146babfb83d5c2b47e971edb43b4566f010577a0ee7d9f4"
)

// WatcherClient provide APIs which support others to subscribe to various event from l2geth
type WatcherClient struct {
	ctx context.Context
	event.Feed

	*ethclient.Client

	orm database.OrmFactory

	confirmations       uint64
	proofGenerationFreq uint64
	skippedOpcodes      map[string]struct{}
	messengerAddress    common.Address
	messengerABI        *abi.ABI

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64

	stopped uint64
	stopCh  chan struct{}
}

// NewL2WatcherClient take a l2geth instance to generate a l2watcherclient instance
func NewL2WatcherClient(ctx context.Context, client *ethclient.Client, confirmations uint64, proofGenFreq uint64, skippedOpcodes map[string]struct{}, messengerAddress common.Address, messengerABI *abi.ABI, orm database.OrmFactory) *WatcherClient {
	savedHeight, err := orm.GetLayer2LatestWatchedHeight()
	if err != nil {
		log.Warn("fetch height from db failed", "err", err)
		savedHeight = 0
	}

	return &WatcherClient{
		ctx:                 ctx,
		Client:              client,
		orm:                 orm,
		processedMsgHeight:  uint64(savedHeight),
		confirmations:       confirmations,
		proofGenerationFreq: proofGenFreq,
		skippedOpcodes:      skippedOpcodes,
		messengerAddress:    messengerAddress,
		messengerABI:        messengerABI,
		stopCh:              make(chan struct{}),
		stopped:             0,
	}
}

// Start the Listening process
func (w *WatcherClient) Start() {
	go func() {
		if w.orm == nil {
			panic("must run L2 watcher with DB")
		}

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
					log.Error("Failed to get_BlockNumber", "err", err)
					continue
				}
				if err := w.tryFetchRunningMissingBlocks(w.ctx, number); err != nil {
					log.Error("Failed to fetchRunningMissingBlocks", "err", err)
				}

				// @todo handle error
				if err := w.fetchContractEvent(number); err != nil {
					log.Error("Failed to fetchContractEvent", "err", err)
				}

				if err := w.tryProposeBatch(); err != nil {
					log.Error("Failed to tryProposeBatch", "err", err)
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
	// Don't use "block_result" table "content" column's BlockTrace.Number,
	// because it might be empty if the corresponding rollup_result is finalized/finalization_skipped
	heightInDB, err := w.orm.GetBlockResultsLatestHeight()
	if err != nil {
		return fmt.Errorf("Failed to GetBlockResults in DB: %v", err)
	}
	backTrackTo := uint64(0)
	if heightInDB > 0 {
		backTrackTo = uint64(heightInDB)
	}

	// start backtracking

	traces := []*types.BlockResult{}
	for number := backTrackFrom; number > backTrackTo; number-- {
		header, err2 := w.HeaderByNumber(ctx, big.NewInt(int64(number)))
		if err2 != nil {
			return fmt.Errorf("Failed to get HeaderByNumber: %v. number: %v", err2, number)
		}
		trace, err2 := w.GetBlockResultByHash(ctx, header.Hash())
		if err2 != nil {
			return fmt.Errorf("Failed to GetBlockResultByHash: %v. number: %v", err2, number)
		}
		log.Info("Retrieved block result", "height", header.Number, "hash", header.Hash())

		traces = append(traces, trace)

	}
	if len(traces) > 0 {
		if err = w.orm.InsertBlockResults(ctx, traces); err != nil {
			return fmt.Errorf("failed to batch insert BlockResults: %v", err)
		}
	}
	return nil
}

// FetchContractEvent pull latest event logs from given contract address and save in DB
func (w *WatcherClient) fetchContractEvent(blockHeight uint64) error {
	fromBlock := int64(w.processedMsgHeight) + 1
	toBlock := int64(blockHeight) - int64(w.confirmations)

	if toBlock < fromBlock {
		return nil
	}

	// warning: uint int conversion...
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlock), // inclusive
		ToBlock:   big.NewInt(toBlock),   // inclusive
		Addresses: []common.Address{
			w.messengerAddress,
		},
		Topics: make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 1)
	query.Topics[0][0] = common.HexToHash(sentMessageEventSignature)

	logs, err := w.FilterLogs(w.ctx, query)
	if err != nil {
		log.Error("Failed to get event logs", "err", err)
		return err
	}
	if len(logs) == 0 {
		return nil
	}
	log.Info("Received new L2 messages", "fromBlock", fromBlock, "toBlock", toBlock,
		"cnt", len(logs))

	eventLogs, err := parseBridgeEventLogs(logs, w.messengerABI)
	if err != nil {
		log.Error("Failed to parse emitted event log", "err", err)
		return err
	}

	err = w.orm.SaveLayer2Messages(w.ctx, eventLogs)
	if err == nil {
		w.processedMsgHeight = uint64(toBlock)
	}
	return err
}

func parseBridgeEventLogs(logs []types.Log, messengerABI *abi.ABI) ([]*orm.Layer2Message, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var parsedlogs []*orm.Layer2Message
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
			log.Error("Failed to unpack layer2 SentMessage event", "err", err)
			return parsedlogs, err
		}
		// target is in topics[1]
		event.Target = common.HexToAddress(vLog.Topics[1].String())
		parsedlogs = append(parsedlogs, &orm.Layer2Message{
			Nonce:      event.MessageNonce.Uint64(),
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
	}

	return parsedlogs, nil
}

// batch-related config
var (
	batchTimeSec      = uint64(5 * 60)  // 5min
	batchGasThreshold = uint64(3000000) // 3M
	batchBlocksLimit  = uint64(10)
)

// TODO:
// + generate batch parallelly
// + TraceHasUnsupportedOpcodes
// + proofGenerationFreq
func (w *WatcherClient) tryProposeBatch() error {
	blocks, err := w.orm.GetBlockInfos(
		map[string]interface{}{"batch_id": sql.NullInt64{Valid: false}},
		fmt.Sprintf("order by number DESC LIMIT %s", batchBlocksLimit),
	)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return nil
	}

	toBatch := []uint64{}
	gasUsed := uint64(0)
	for _, block := range blocks {
		gasUsed += block.GasUsed
		if gasUsed > batchGasThreshold {
			break
		}

		toBatch = append(toBatch, block.Number)
	}

	if gasUsed < batchGasThreshold && blocks[0].BlockTimestamp+batchTimeSec < uint64(time.Now().Unix()) {
		return nil
	}

	// keep gasUsed below threshold
	if len(toBatch) >= 2 {
		gasUsed -= blocks[len(toBatch)-1].GasUsed
		toBatch = toBatch[:len(toBatch)-1]
	}

	parents, err := w.orm.GetBlockInfos(map[string]interface{}{"numer": toBatch[0].Number - 1})
	if err != nil || len(parents) == 0 {
		return errors.New("Cannot find last batch's end_block")
	}

	return w.createBatchForBlocks(toBatch, parents[0].Hash, gasUsed)
}

func (w *WatcherClient) createBatchForBlocks(blocks []uint64, parent_hash string, gasUsed uint64) error {
	dbTx, err := w.orm.Beginx()
	if err != nil {
		return err
	}

	var dbTxErr error
	defer func() {
		if dbTxErr != nil {
			if err := dbTx.Rollback(); err != nil {
				log.Error("dbTx.Rollback()", "err", err)
			}
		}
	}()

	var batchID uint64
	batchID, dbTxErr = w.orm.NewBatchInDBTx(dbTx, parent_hash, gasUsed)
	if dbTxErr != nil {
		return dbTxErr
	}

	if dbTxErr = w.orm.SetBatchIDForBlocksInDBTx(dbTx, blocks, batchID); dbTxErr != nil {
		return dbTxErr
	}

	dbTxErr = dbTx.Commit()
	return dbTxErr
}
