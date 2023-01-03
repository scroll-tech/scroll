package l1

import (
	"context"

	"github.com/scroll-tech/go-ethereum/ethclient"

	"scroll-tech/common/viper"
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
func New(ctx context.Context, vp *viper.Viper, orm database.OrmFactory) (*Backend, error) {
	client, err := ethclient.Dial(vp.GetString("endpoint"))
	if err != nil {
		return nil, err
	}

	relayer, err := NewLayer1Relayer(ctx, client, orm, vp.Sub("relayer_config"))
	if err != nil {
		return nil, err
	}

	watcher := NewWatcher(ctx, client, vp, orm)

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
