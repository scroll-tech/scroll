package l2

import (
	"context"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/viper"
	"scroll-tech/database"
)

// Backend manage the resources and services of L2 backend.
// The backend should monitor events in layer 2 and relay transactions to layer 1
type Backend struct {
	l2Watcher *WatcherClient
	relayer   *Layer2Relayer
	orm       database.OrmFactory
}

// New returns a new instance of Backend.
func New(ctx context.Context, vp *viper.Viper, orm database.OrmFactory) (*Backend, error) {
	client, err := ethclient.Dial(vp.GetString("endpoint"))
	if err != nil {
		return nil, err
	}

	relayer, err := NewLayer2Relayer(ctx, client, orm, vp.Sub("relayer_config"))
	if err != nil {
		return nil, err
	}

	l2Watcher := NewL2WatcherClient(ctx, client, vp, orm)

	return &Backend{
		l2Watcher: l2Watcher,
		relayer:   relayer,
		orm:       orm,
	}, nil
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
