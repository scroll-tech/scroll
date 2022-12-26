package l1

import (
	"context"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/spf13/viper"

	"scroll-tech/database"
)

// Backend manage the resources and services of L1 backend.
// The backend should monitor events in layer 1 and relay transactions to layer 2
type Backend struct {
	watcher *Watcher
	relayer *Layer1Relayer
	orm     database.OrmFactory
}

// New returns a new instance of Backend.
func New(ctx context.Context, orm database.OrmFactory) (*Backend, error) {
	client, err := ethclient.Dial(viper.GetString("db_config.endpoint"))
	if err != nil {
		return nil, err
	}

	relayer, err := NewLayer1Relayer(ctx, client, orm, viper.Sub("l1_config.relayer_config"))
	if err != nil {
		return nil, err
	}

	watcher := NewWatcher(ctx, client, orm)

	return &Backend{
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
