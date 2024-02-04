package watcher

import (
	"context"
	"fmt"
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	rollupTypes "github.com/scroll-tech/go-ethereum/rollup/types"
	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	bridgeAbi "scroll-tech/rollup/abi"
	"scroll-tech/rollup/internal/orm"
)

// L2WatcherClient provide APIs which support others to subscribe to various event from l2geth
type L2WatcherClient struct {
	ctx context.Context
	event.Feed

	*ethclient.Client

	l2BlockOrm *orm.L2Block

	confirmations rpc.BlockNumber

	messageQueueAddress  common.Address
	messageQueueABI      *abi.ABI
	withdrawTrieRootSlot common.Hash

	metrics *l2WatcherMetrics
}

// NewL2WatcherClient take a l2geth instance to generate a l2watcherclient instance
func NewL2WatcherClient(ctx context.Context, client *ethclient.Client, confirmations rpc.BlockNumber, messageQueueAddress common.Address, withdrawTrieRootSlot common.Hash, db *gorm.DB, reg prometheus.Registerer) *L2WatcherClient {
	return &L2WatcherClient{
		ctx:    ctx,
		Client: client,

		l2BlockOrm: orm.NewL2Block(db),

		confirmations: confirmations,

		messageQueueAddress:  messageQueueAddress,
		messageQueueABI:      bridgeAbi.L2MessageQueueABI,
		withdrawTrieRootSlot: withdrawTrieRootSlot,

		metrics: initL2WatcherMetrics(reg),
	}
}

const blocksFetchLimit = uint64(10)

// TryFetchRunningMissingBlocks attempts to fetch and store block traces for any missing blocks.
func (w *L2WatcherClient) TryFetchRunningMissingBlocks(blockHeight uint64) {
	w.metrics.fetchRunningMissingBlocksTotal.Inc()
	heightInDB, err := w.l2BlockOrm.GetL2BlocksLatestHeight(w.ctx)
	if err != nil {
		log.Error("failed to GetL2BlocksLatestHeight", "err", err)
		return
	}

	// Fetch and store block traces for missing blocks
	for from := heightInDB + 1; from <= blockHeight; from += blocksFetchLimit {
		to := from + blocksFetchLimit - 1

		if to > blockHeight {
			to = blockHeight
		}

		if err = w.getAndStoreBlocks(w.ctx, from, to); err != nil {
			log.Error("fail to getAndStoreBlocks", "from", from, "to", to, "err", err)
			return
		}
		w.metrics.fetchRunningMissingBlocksHeight.Set(float64(to))
		w.metrics.rollupL2BlocksFetchedGap.Set(float64(blockHeight - to))
	}
}

func (w *L2WatcherClient) getAndStoreBlocks(ctx context.Context, from, to uint64) error {
	var blocks []*rollupTypes.WrappedBlock
	for number := from; number <= to; number++ {
		log.Debug("retrieving block", "height", number)
		block, err := w.GetBlockByNumberOrHash(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(number)))
		if err != nil {
			return fmt.Errorf("failed to GetBlockByNumberOrHash: %v. number: %v", err, number)
		}
		if block.RowConsumption == nil {
			return fmt.Errorf("fetched block does not contain RowConsumption. number: %v", number)
		}

		log.Info("retrieved block", "height", block.Header().Number, "hash", block.Header().Hash().String())

		withdrawRoot, err3 := w.StorageAt(ctx, w.messageQueueAddress, w.withdrawTrieRootSlot, big.NewInt(int64(number)))
		if err3 != nil {
			return fmt.Errorf("failed to get withdrawRoot: %v. number: %v", err3, number)
		}
		blocks = append(blocks, &rollupTypes.WrappedBlock{
			Header:         block.Header(),
			Transactions:   block.Transactions(),
			WithdrawRoot:   common.BytesToHash(withdrawRoot),
			RowConsumption: block.RowConsumption,
		})
	}

	if len(blocks) > 0 {
		for _, block := range blocks {
			w.metrics.rollupL2BlockL1CommitCalldataSize.Set(float64(block.EstimateL1CommitCalldataSize()))
		}
		if err := w.l2BlockOrm.InsertL2Blocks(w.ctx, blocks); err != nil {
			return fmt.Errorf("failed to batch insert BlockTraces: %v", err)
		}
	}

	return nil
}
