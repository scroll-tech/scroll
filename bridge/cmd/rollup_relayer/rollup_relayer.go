package rolluprelayer

import (
	"context"
	"time"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/relayer"
	"scroll-tech/bridge/utils"
)

// L2RollupRelayer is struct to wrap l2.relayer to transmit rollup msg
type L2RollupRelayer struct {
	ctx     context.Context
	cancel  context.CancelFunc
	relayer *relayer.Layer2Relayer
	stopCh  chan struct{}
	db      orm.L2MessageOrm
}

// NewL2RollupRelayer creates a new instance of L2RollupRelayer
func NewL2RollupRelayer(ctx context.Context, cfg *config.RelayerConfig, db database.OrmFactory) (*L2RollupRelayer, error) {
	msgRelayer, err := relayer.NewLayer2Relayer(ctx, db, cfg)
	if err != nil {
		return nil, err
	}
	subCtx, cancel := context.WithCancel(ctx)
	return &L2RollupRelayer{
		ctx:     subCtx,
		cancel:  cancel,
		relayer: msgRelayer,
		db:      db,
		stopCh:  make(chan struct{}),
	}, nil
}

// Start process
func (r *L2RollupRelayer) Start() {
	go func() {
		ctx, cancel := context.WithCancel(r.ctx)
		// trigger by timer
		tk_pending := time.NewTicker(time.Second)
		tk_commit := time.NewTicker(time.Second)
		defer cancel()

		go utils.Loop(ctx, tk_pending, r.relayer.ProcessPendingBatches)
		go utils.Loop(ctx, tk_commit, r.relayer.ProcessCommittedBatches)

		for {
			select {
			case confirmation := <-r.relayer.GetRollupCh():
				r.relayer.HandleConfirmation(confirmation)
			case <-r.ctx.Done():
				tk_pending.Stop()
				tk_commit.Stop()
				return
			}
		}
	}()
}

// Stop sends the stop signal to stop chan
func (r *L2RollupRelayer) Stop() {
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
}
