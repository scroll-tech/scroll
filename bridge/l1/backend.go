package l1

import (
	"context"

	"github.com/scroll-tech/go-ethereum/ethclient"

	"scroll-tech/database"

	"scroll-tech/bridge/config"
)

// Backend manage the resources and services of L1 backend.
// The backend should monitor events in layer 1 and relay transactions to layer 2
type Backend struct {
	cfg     *config.L1Config
	watcher *Watcher
	relayer *Layer1Relayer
	orm     database.OrmFactory
}

// New returns a new instance of Backend.
func New(ctx context.Context, cfg *config.L1Config, orm database.OrmFactory) (*Backend, error) {
	client, err := ethclient.Dial(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	relayer, err := NewLayer1Relayer(ctx, client, int64(cfg.Confirmations), orm, cfg.RelayerConfig)
	if err != nil {
		return nil, err
	}

	watcher, err := NewWatcher(ctx, client, cfg.StartHeight, cfg.Confirmations, cfg.L1MessengerAddress, cfg.L1MessageQueueAddress, cfg.RollupContractAddress, orm)
	if err != nil {
		return nil, err
	}

	return &Backend{
		cfg:     cfg,
		watcher: watcher,
		relayer: relayer,
		orm:     orm,
	}, nil
}

// Start Backend module.
func (l1 *Backend) Start() error {
	l1.watcher.Start()
	l1.relayer.Start()
	return nil
}

// Stop Backend module.
func (l1 *Backend) Stop() {
	l1.watcher.Stop()
	l1.relayer.Stop()
}
