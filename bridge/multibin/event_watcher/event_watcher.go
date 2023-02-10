package eventwatcher

import (
	"context"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"
	"scroll-tech/bridge/l2"
)

// L1EventWatcher is sturct to wrap l1.watcher
type L1EventWatcher struct {
	ctx     context.Context
	watcher *l1.Watcher
	client  *ethclient.Client
	stopCh  chan struct{}
}

// L2EventWatcher is struct to wrap l2.watcher
type L2EventWatcher struct {
	ctx           context.Context
	watcher       *l2.WatcherClient
	client        *ethclient.Client
	confirmations uint64
	stopCh        chan struct{}
}

// NewL2EventWatcher creates a new instance of L2EventWatcher
func NewL2EventWatcher(ctx context.Context, client *ethclient.Client, cfg *config.L2Config, db database.OrmFactory) *L2EventWatcher {
	watcher := l2.NewL2WatcherClient(ctx, client, cfg.Confirmations, cfg.BatchProposerConfig, cfg.RelayerConfig.MessengerContractAddress, db)
	return &L2EventWatcher{
		ctx:           ctx,
		watcher:       watcher,
		client:        client,
		confirmations: cfg.Confirmations.Number,
		stopCh:        make(chan struct{}),
	}
}

// NewL1EventWatcher creates a new instance of L1EventWatcher
func NewL1EventWatcher(ctx context.Context, client *ethclient.Client, cfg *config.L1Config, db database.OrmFactory) *L1EventWatcher {
	watcher := l1.NewWatcher(ctx, client, cfg.StartHeight, cfg.Confirmations, cfg.L1MessengerAddress, cfg.RollupContractAddress, db)
	return &L1EventWatcher{
		ctx:     ctx,
		watcher: watcher,
		client:  client,
		stopCh:  make(chan struct{}),
	}
}

// Start runs go routine to fetch contract events on L1
func (l1w *L1EventWatcher) Start() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for ; true; <-ticker.C {
			select {
			case <-l1w.stopCh:
				return

			default:
				blockNumber, err := l1w.client.BlockNumber(l1w.ctx)
				if err != nil {
					log.Error("Failed to get block number", "err", err)
					continue
				}
				if err := l1w.watcher.FetchContractEvent(blockNumber); err != nil {
					log.Error("Failed to fetch bridge contract", "err", err)
				}
			}
		}
	}()
}

// Stop sends the stop signal to stop chan
func (l1w *L1EventWatcher) Stop() {
	close(l1w.stopCh)
}

// Start runs go routine to fetch contract events on L2
func (l2w *L2EventWatcher) Start() {
	// event fetcher loop
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-l2w.stopCh:
				return

			case <-ticker.C:
				// get current height
				number, err := l2w.client.BlockNumber(l2w.ctx)
				if err != nil {
					log.Error("failed to get_BlockNumber", "err", err)
					continue
				}

				if number >= l2w.confirmations {
					number = number - l2w.confirmations
				} else {
					number = 0
				}

				l2w.watcher.FetchContractEvent(number)
			}
		}
	}()
}

// Stop sends the stop signal to stop chan
func (l2w *L2EventWatcher) Stop() {
	close(l2w.stopCh)
}
