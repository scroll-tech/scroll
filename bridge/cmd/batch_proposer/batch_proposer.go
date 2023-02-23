package batchproposer

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
	bp "scroll-tech/bridge/watcher/batch_proposer"
)

// L2BatchPropser is struct to wrap l2.watcher
type L2BatchPropser struct {
	ctx    context.Context
	cancel context.CancelFunc

	watcher       *watcher.WatcherClient
	client        *ethclient.Client
	confirmations rpc.BlockNumber
	batchProposer *bp.BatchProposer
}

// NewL2BatchPropser creates a new instance of L2BatchPropser
func NewL2BatchProposer(ctx context.Context, client *ethclient.Client, cfg *config.L2Config, db database.OrmFactory) (*L2BatchPropser, error) {
	subCtx, cancel := context.WithCancel(ctx)
	watcher := watcher.NewL2WatcherClient(ctx, client, cfg.Confirmations, cfg.BatchProposerConfig, cfg.RelayerConfig.MessengerContractAddress, db)
	return &L2BatchPropser{
		ctx:           subCtx,
		cancel:        cancel,
		watcher:       watcher,
		client:        client,
		confirmations: cfg.Confirmations,
		batchProposer: watcher.GetBatchProposer(),
	}, nil
}

// Start runs go routine to fetch contract events on L2
func (b *L2BatchPropser) Start() {
	// Todo: Refactoring this process
	go utils.LoopWithContext(b.ctx, time.NewTicker(3*time.Second), func(ctx context.Context) {
		number, err := utils.GetLatestConfirmedBlockNumber(ctx, b.client, b.confirmations)
		if err != nil {
			log.Error("failed to get block number", "err", err)
			return
		}
		b.watcher.TryFetchRunningMissingBlocks(ctx, number)
	})

	// batch proposer loop
	go utils.Loop(b.ctx, time.NewTicker(3*time.Second), b.batchProposer.TryProposeBatch)
}

// Stop sends the stop signal to stop chan
func (b *L2BatchPropser) Stop() {
	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}
}
