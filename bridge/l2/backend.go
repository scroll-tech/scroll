package l2

import (
	"context"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/bridge"

	"scroll-tech/database"

	"scroll-tech/bridge/config"
)

// Backend manage the resources and services of L2 backend.
// The backend should monitor events in layer 2 and relay transactions to layer 1
type Backend struct {
	cfg       *config.L2Config
	client    *ethclient.Client
	l2Watcher *WatcherClient
	relayer   *Layer2Relayer
	orm       database.OrmFactory
}

// New returns a new instance of Backend.
func New(ctx context.Context, cfg *config.L2Config, orm database.OrmFactory, l1Backend bridge.API) (*Backend, error) {
	client, err := ethclient.Dial(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	relayer, err := NewLayer2Relayer(ctx, client, l1Backend.GetClient(), cfg.ProofGenerationFreq, cfg.SkippedOpcodes, orm, cfg.RelayerConfig)
	if err != nil {
		return nil, err
	}

	l2Watcher := NewL2WatcherClient(ctx, client, cfg.Confirmations, cfg.ProofGenerationFreq, cfg.SkippedOpcodes, cfg.L2MessengerAddress, orm)

	return &Backend{
		cfg:       cfg,
		client:    client,
		l2Watcher: l2Watcher,
		relayer:   relayer,
		orm:       orm,
	}, nil
}

// GetClient return l2 chain client instance.
func (l2 *Backend) GetClient() *ethclient.Client {
	return l2.client
}

// Start Backend module.
func (l2 *Backend) Start() error {
	l2.l2Watcher.Start()
	l2.relayer.Start()
	return nil
}

// Stop Backend module.
func (l2 *Backend) Stop() {
	l2.l2Watcher.Stop()
	l2.relayer.Stop()
}

// APIs collect API modules.
func (l2 *Backend) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "l2",
			Version:   "1.0",
			Service:   WatcherAPI(l2.l2Watcher),
			Public:    true,
		},
	}
}

// MockBlockResult for test case
func (l2 *Backend) MockBlockResult(blockResult *types.BlockResult) {
	l2.l2Watcher.Send(blockResult)
}
