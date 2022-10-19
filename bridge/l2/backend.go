package l2

import (
	"context"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/database"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
)

// Backend manage the resources and services of L2 backend.
// The backend should monitor events in layer 2 and relay transactions to layer 1
type Backend struct {
	cfg       *config.L2Config
	l2Watcher *WatcherClient
	relayer   *Layer2Relayer
	orm       database.OrmFactory
}

// New returns a new instance of Backend.
func New(ctx context.Context, cfg *config.L2Config, orm database.OrmFactory) (*Backend, error) {
	client, err := ethclient.Dial(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	l2MessengerABI, err := bridge_abi.L2MessengerMetaData.GetAbi()
	if err != nil {
		log.Warn("new L2MessengerABI failed", "err", err)
		return nil, err
	}

	skippedOpcodes := make(map[string]struct{}, len(cfg.SkippedOpcodes))
	for _, op := range cfg.SkippedOpcodes {
		skippedOpcodes[op] = struct{}{}
	}

	proofGenerationFreq := cfg.ProofGenerationFreq
	if proofGenerationFreq == 0 {
		log.Warn("receive 0 proof_generation_freq, change to 1")
		proofGenerationFreq = 1
	}

	relayer, err := NewLayer2Relayer(ctx, client, proofGenerationFreq, skippedOpcodes, int64(cfg.Confirmations), orm, cfg.RelayerConfig)
	if err != nil {
		return nil, err
	}

	l2Watcher := NewL2WatcherClient(ctx, client, cfg.Confirmations, proofGenerationFreq, skippedOpcodes, cfg.L2MessengerAddress, l2MessengerABI, orm)

	return &Backend{
		cfg:       cfg,
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

// MockBlockResult for test case
func (l2 *Backend) MockBlockResult(blockResult *types.BlockResult) {
	l2.l2Watcher.Send(blockResult)
}
