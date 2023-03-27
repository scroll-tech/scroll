package l2

import (
	"context"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/database"

	"scroll-tech/bridge/config"
)

// Backend manage the resources and services of L2 backend.
// The backend should monitor events in layer 2 and relay transactions to layer 1
type Backend struct {
	cfg           *config.L2Config
	watcher       *WatcherClient
	relayer       *Layer2Relayer
	batchProposer *BatchProposer
	orm           database.OrmFactory
}

// New returns a new instance of Backend.
func New(ctx context.Context, cfg *config.L2Config, orm database.OrmFactory) (*Backend, error) {
	client, err := ethclient.Dial(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	// Note: initialize watcher before relayer to keep DB consistent.
	// Otherwise, there will be a race condition between watcher.initializeGenesis and relayer.ProcessPendingBatches.
	watcher := NewL2WatcherClient(ctx, client, cfg.Confirmations, cfg.L2MessengerAddress, cfg.L2MessageQueueAddress, cfg.WithdrawTrieRootSlot, orm)

	proposer := NewBatchProposer(ctx, cfg.BatchProposerConfig, orm)
	relayer, err := NewLayer2Relayer(ctx, client, orm, cfg.RelayerConfig)
	if err != nil {
		return nil, err
	}
	proposer.SetLayer2Relayer(relayer)
	relayer.SetBatchProposer(proposer)

	return &Backend{
		cfg:           cfg,
		watcher:       watcher,
		relayer:       relayer,
		batchProposer: proposer,
		orm:           orm,
	}, nil
}

// Start Backend module.
func (l2 *Backend) Start() error {
	l2.watcher.Start()
	l2.relayer.Start()
	l2.batchProposer.Start()
	return nil
}

// Stop Backend module.
func (l2 *Backend) Stop() {
	l2.batchProposer.Stop()
	l2.relayer.Stop()
	l2.watcher.Stop()
}

// APIs collect API modules.
func (l2 *Backend) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "l2",
			Version:   "1.0",
			Service:   WatcherAPI(l2.watcher),
			Public:    true,
		},
	}
}
