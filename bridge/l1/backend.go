package l1

import (
	"context"

	"github.com/scroll-tech/go-ethereum/ethclient"

	"scroll-tech/database/orm"

	"scroll-tech/bridge"

	"scroll-tech/bridge/config"
)

// Backend manage the resources and services of L1 backend.
// The backend should monitor events in layer 1 and relay transactions to layer 2
type Backend struct {
	cfg     *config.L1Config
	client  *ethclient.Client
	watcher *Watcher
	relayer *Layer1Relayer
	orm     orm.Layer1MessageOrm
}

// New returns a new instance of Backend.
func New(ctx context.Context, cfg *config.L1Config, orm orm.Layer1MessageOrm, l2Backend bridge.API) (*Backend, error) {
	client, err := ethclient.Dial(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	relayer, err := NewLayer1Relayer(ctx, l2Backend.GetClient(), cfg.RelayerConfig, orm)
	if err != nil {
		return nil, err
	}

	watcher := NewWatcher(ctx, client, cfg.StartHeight, cfg.Confirmations, cfg.L1MessengerAddress, orm)

	return &Backend{
		cfg:     cfg,
		client:  client,
		watcher: watcher,
		relayer: relayer,
		orm:     orm,
	}, nil
}

// GetClient return l1 chain client instance.
func (l1 *Backend) GetClient() *ethclient.Client {
	return l1.client
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
