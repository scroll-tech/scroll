package batchproposer

import (
	"context"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l2"
	"scroll-tech/bridge/utils"
	"scroll-tech/database"
)

// L2BatchPropser is struct to wrap l2.watcher
type L2BatchPropser struct {
	ctx           context.Context
	watcher       *l2.WatcherClient
	client        *ethclient.Client
	confirmations utils.ConfirmationParams
	batchProposer *l2.BatchProposer
	stopCh        chan struct{}
}

// NewL2BatchPropser creates a new instance of L2BatchPropser
func NewL2BatchPropser(ctx context.Context, client *ethclient.Client, cfg *config.L2Config, db database.OrmFactory) (*L2BatchPropser, error) {
	watcher := l2.NewL2WatcherClient(ctx, client, cfg.Confirmations, cfg.BatchProposerConfig, cfg.RelayerConfig.MessengerContractAddress, db)
	return &L2BatchPropser{
		ctx:           ctx,
		watcher:       watcher,
		client:        client,
		confirmations: cfg.Confirmations,
		batchProposer: watcher.GetBatchProposer(),
		stopCh:        make(chan struct{}),
	}, nil
}

// Start runs go routine to fetch contract events on L2
func (b *L2BatchPropser) Start() {
	// Todo: Refactoring this process
	go func() {
		ctx, cancel := context.WithCancel(b.ctx)
		// trace fetcher loop
		go func(ctx context.Context) {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return

				case <-ticker.C:
					number, err := utils.GetLatestConfirmedBlockNumber(ctx, b.client, b.confirmations)
					if err != nil {
						log.Error("failed to get block number", "err", err)
						continue
					}
					b.watcher.TryFetchRunningMissingBlocks(ctx, number)
				}
			}
		}(ctx)

		// batch proposer loop
		go func(ctx context.Context) {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return

				case <-ticker.C:
					b.batchProposer.TryProposeBatch()
				}
			}
		}(ctx)

		<-b.stopCh
		cancel()

	}()
}

// Stop sends the stop signal to stop chan
func (b *L2BatchPropser) Stop() {
	close(b.stopCh)
}
