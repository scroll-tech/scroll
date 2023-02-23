package eventwatcher

import (
	"context"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/database"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/utils"
	"scroll-tech/bridge/watcher"
)

// L1EventWatcher is sturct to wrap l1.watcher
type L1EventWatcher struct {
	ctx    context.Context
	cancel context.CancelFunc

	watcher *watcher.Watcher
	client  *ethclient.Client
}

// L2EventWatcher is struct to wrap l2.watcher
type L2EventWatcher struct {
	ctx    context.Context
	cancel context.CancelFunc

	watcher       *watcher.WatcherClient
	client        *ethclient.Client
	confirmations rpc.BlockNumber
	stopCh        chan struct{}
}

// NewL2EventWatcher creates a new instance of L2EventWatcher
func NewL2EventWatcher(ctx context.Context, client *ethclient.Client, cfg *config.L2Config, db database.OrmFactory) *L2EventWatcher {
	subCtx, cancel := context.WithCancel(ctx)
	watcher := watcher.NewL2WatcherClient(ctx, client, cfg.Confirmations, cfg.BatchProposerConfig, cfg.RelayerConfig.MessengerContractAddress, db)
	return &L2EventWatcher{
		ctx:           subCtx,
		cancel:        cancel,
		watcher:       watcher,
		client:        client,
		confirmations: cfg.Confirmations,
		stopCh:        make(chan struct{}),
	}
}

// NewL1EventWatcher creates a new instance of L1EventWatcher
func NewL1EventWatcher(ctx context.Context, client *ethclient.Client, cfg *config.L1Config, db database.OrmFactory) *L1EventWatcher {
	subCtx, cancel := context.WithCancel(ctx)
	watcher := watcher.NewWatcher(ctx, client, cfg.StartHeight, cfg.Confirmations, cfg.L1MessengerAddress, cfg.RollupContractAddress, db)
	return &L1EventWatcher{
		ctx:     subCtx,
		cancel:  cancel,
		watcher: watcher,
		client:  client,
	}
}

// Start runs go routine to fetch contract events on L1
func (l1w *L1EventWatcher) Start() {
	go utils.LoopWithContext(l1w.ctx, time.NewTicker(10*time.Second), func(ctx context.Context) {
		blockNumber, err := l1w.client.BlockNumber(ctx)
		if err != nil {
			log.Error("Failed to get block number", "err", err)
			return
		}
		if err := l1w.watcher.FetchContractEvent(blockNumber); err != nil {
			log.Error("Failed to fetch bridge contract", "err", err)
		}
	})
}

// Stop sends the stop signal to stop chan
func (l1w *L1EventWatcher) Stop() {
	if l1w.cancel != nil {
		l1w.cancel()
		l1w.cancel = nil
	}
}

// Start runs go routine to fetch contract events on L2
func (l2w *L2EventWatcher) Start() {
	// event fetcher loop
	go utils.LoopWithContext(l2w.ctx, time.NewTicker(3*time.Second), func(ctx context.Context) {
		number, err := utils.GetLatestConfirmedBlockNumber(ctx, l2w.client, l2w.confirmations)
		if err != nil {
			log.Error("failed to get block number", "err", err)
			return
		}

		l2w.watcher.FetchContractEvent(number)
	})
}

// Stop sends the stop signal to stop chan
func (l2w *L2EventWatcher) Stop() {
	if l2w.cancel != nil {
		l2w.cancel()
		l2w.cancel = nil
	}
}
