package l1

import (
	"context"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database/orm"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
)

// Backend manage the resources and services of L1 backend.
// The backend should monitor events in layer 1 and relay transactions to layer 2
type Backend struct {
	cfg     *config.L1Config
	watcher *Watcher
	relayer *Layer1Relayer
	orm     orm.L1MessageOrm
}

// New returns a new instance of Backend.
func New(ctx context.Context, cfg *config.L1Config, orm orm.L1MessageOrm) (*Backend, error) {
	client, err := ethclient.Dial(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	l1MessengerABI, err := bridge_abi.L1MessengerMetaData.GetAbi()
	if err != nil {
		log.Warn("new L1MessengerABI failed", "err", err)
		return nil, err
	}

	relayer, err := NewLayer1Relayer(ctx, client, int64(cfg.Confirmations), orm, cfg.RelayerConfig)
	if err != nil {
		return nil, err
	}

	watcher := NewWatcher(ctx, client, cfg.StartHeight, cfg.Confirmations, cfg.L1MessengerAddress, l1MessengerABI, orm)

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
