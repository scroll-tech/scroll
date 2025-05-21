package watcher

import (
	"context"
	"fmt"
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

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
	withdrawTrieRootSlot common.Hash

	metrics *l2WatcherMetrics

	chainCfg *params.ChainConfig
}

// NewL2WatcherClient take a l2geth instance to generate a l2watcherclient instance
func NewL2WatcherClient(ctx context.Context, client *ethclient.Client, confirmations rpc.BlockNumber, messageQueueAddress common.Address, withdrawTrieRootSlot common.Hash, chainCfg *params.ChainConfig, db *gorm.DB, reg prometheus.Registerer) *L2WatcherClient {
	return &L2WatcherClient{
		ctx:    ctx,
		Client: client,

		l2BlockOrm: orm.NewL2Block(db),

		confirmations: confirmations,

		messageQueueAddress:  messageQueueAddress,
		withdrawTrieRootSlot: withdrawTrieRootSlot,

		metrics: initL2WatcherMetrics(reg),

		chainCfg: chainCfg,
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
			log.Error("fail to getAndStoreBlockTraces", "from", from, "to", to, "err", err)
			return
		}
		w.metrics.fetchRunningMissingBlocksHeight.Set(float64(to))
		w.metrics.rollupL2BlocksFetchedGap.Set(float64(blockHeight - to))
	}
}

func (w *L2WatcherClient) getAndStoreBlocks(ctx context.Context, from, to uint64) error {
	var blocks []*encoding.Block
	for number := from; number <= to; number++ {
		log.Debug("retrieving block", "height", number)
		block, err := w.GetBlockByNumberOrHash(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(number)))
		if err != nil {
			return fmt.Errorf("failed to GetBlockByNumberOrHash: %v. number: %v", err, number)
		}

		var count int
		for _, tx := range block.Transactions() {
			if tx.IsL1MessageTx() {
				count++
			}
		}
		log.Info("retrieved block", "height", block.Header().Number, "hash", block.Header().Hash().String(), "L1 message count", count)

		withdrawRoot, err3 := w.StorageAt(ctx, w.messageQueueAddress, w.withdrawTrieRootSlot, big.NewInt(int64(number)))
		if err3 != nil {
			return fmt.Errorf("failed to get withdrawRoot: %v. number: %v", err3, number)
		}
		blocks = append(blocks, &encoding.Block{
			Header:       block.Header(),
			Transactions: encoding.TxsToTxsData(block.Transactions()),
			WithdrawRoot: common.BytesToHash(withdrawRoot),
		})
	}

	if len(blocks) > 0 {
		for _, block := range blocks {
			codec := encoding.CodecFromConfig(w.chainCfg, block.Header.Number, block.Header.Time)
			if codec == nil {
				return fmt.Errorf("failed to retrieve codec for block number %v and time %v", block.Header.Number, block.Header.Time)
			}
			w.metrics.rollupL2WatcherSyncThroughput.Add(float64(block.Header.GasUsed))
		}
		if err := w.l2BlockOrm.InsertL2Blocks(w.ctx, blocks); err != nil {
			return fmt.Errorf("failed to batch insert BlockTraces: %v", err)
		}
	}

	return nil
}
