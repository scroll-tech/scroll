package watcher

import (
	"context"
	"fmt"
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
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
	rpcCli *rpc.Client

	l2BlockOrm *orm.L2Block

	confirmations rpc.BlockNumber

	messageQueueAddress  common.Address
	withdrawTrieRootSlot common.Hash

	validiumMode bool

	metrics *l2WatcherMetrics

	chainCfg *params.ChainConfig
}

// NewL2WatcherClient take a l2geth instance to generate a l2watcherclient instance
func NewL2WatcherClient(ctx context.Context, client *rpc.Client, confirmations rpc.BlockNumber, messageQueueAddress common.Address, withdrawTrieRootSlot common.Hash, chainCfg *params.ChainConfig, db *gorm.DB, validiumMode bool, reg prometheus.Registerer) *L2WatcherClient {
	w := &L2WatcherClient{
		ctx:    ctx,
		Client: ethclient.NewClient(client),
		rpcCli: client,

		l2BlockOrm: orm.NewL2Block(db),

		confirmations: confirmations,

		messageQueueAddress:  messageQueueAddress,
		withdrawTrieRootSlot: withdrawTrieRootSlot,

		validiumMode: validiumMode,

		metrics: initL2WatcherMetrics(reg),

		chainCfg: chainCfg,
	}

	// Seed the gauge from the latest stored height instead of starting from 0.
	if latestHeight, err := w.l2BlockOrm.GetL2BlocksLatestHeight(ctx); err != nil {
		log.Warn("failed to initialize l2 watcher fetched height metric", "err", err)
	} else {
		w.metrics.fetchRunningMissingBlocksHeight.Set(float64(latestHeight))
	}

	return w
}

const blocksFetchLimit = uint64(10)

// TryFetchRunningMissingBlocks attempts to fetch and store block traces for any missing blocks.
func (w *L2WatcherClient) TryFetchRunningMissingBlocks(blockHeight uint64) error {
	w.metrics.fetchRunningMissingBlocksTotal.Inc()
	heightInDB, err := w.l2BlockOrm.GetL2BlocksLatestHeight(w.ctx)
	if err != nil {
		log.Error("failed to GetL2BlocksLatestHeight", "err", err)
		return fmt.Errorf("failed to GetL2BlocksLatestHeight: %w", err)
	}

	// Fetch and store block traces for missing blocks
	for from := heightInDB + 1; from <= blockHeight; from += blocksFetchLimit {
		to := from + blocksFetchLimit - 1

		if to > blockHeight {
			to = blockHeight
		}

		if err = w.GetAndStoreBlocks(w.ctx, from, to); err != nil {
			log.Error("fail to getAndStoreBlockTraces", "from", from, "to", to, "err", err)
			return fmt.Errorf("fail to getAndStoreBlockTraces: %w", err)
		}
		w.metrics.fetchRunningMissingBlocksHeight.Set(float64(to))
		w.metrics.rollupL2BlocksFetchedGap.Set(float64(blockHeight - to))
	}

	return nil
}

func (w *L2WatcherClient) GetAndStoreBlocks(ctx context.Context, from, to uint64) error {
	var blocks []*encoding.Block
	for number := from; number <= to; number++ {
		log.Debug("retrieving block", "height", number)
		block, err := w.BlockByNumber(ctx, new(big.Int).SetUint64(number))
		if err != nil {
			return fmt.Errorf("failed to BlockByNumber: %v. number: %v", err, number)
		}

		blockTxs := block.Transactions()

		var count int
		for _, tx := range blockTxs {
			if tx.IsL1MessageTx() {
				count++
			}
		}
		log.Info("retrieved block", "height", block.Header().Number, "hash", block.Header().Hash().String(), "L1 message count", count)

		// use original (encrypted) L1 message txs in validium mode
		if w.validiumMode {
			var txs []*types.Transaction

			if count > 0 {
				log.Info("Fetching encrypted messages in validium mode")
				err = w.rpcCli.CallContext(ctx, &txs, "scroll_getL1MessagesInBlock", block.Hash(), "synced")
				if err != nil {
					return fmt.Errorf("failed to get L1 messages: %v, block hash: %v", err, block.Hash().Hex())
				}
			}

			// sanity check
			if len(txs) != count {
				return fmt.Errorf("L1 message count mismatch: expected %d, got %d", count, len(txs))
			}

			for ii := 0; ii < count; ii++ {
				// sanity check
				if blockTxs[ii].AsL1MessageTx().QueueIndex != txs[ii].AsL1MessageTx().QueueIndex {
					return fmt.Errorf("L1 message queue index mismatch at index %d: expected %d, got %d", ii, blockTxs[ii].AsL1MessageTx().QueueIndex, txs[ii].AsL1MessageTx().QueueIndex)
				}

				log.Info("Replacing L1 message tx in validium mode", "index", ii, "queueIndex", txs[ii].AsL1MessageTx().QueueIndex, "decryptedTxHash", blockTxs[ii].Hash().Hex(), "originalTxHash", txs[ii].Hash().Hex())
				blockTxs[ii] = txs[ii]
			}
		}

		withdrawRoot, err3 := w.StorageAt(ctx, w.messageQueueAddress, w.withdrawTrieRootSlot, big.NewInt(int64(number)))
		if err3 != nil {
			return fmt.Errorf("failed to get withdrawRoot: %v. number: %v", err3, number)
		}
		blocks = append(blocks, &encoding.Block{
			Header:       block.Header(),
			Transactions: encoding.TxsToTxsData(blockTxs),
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
