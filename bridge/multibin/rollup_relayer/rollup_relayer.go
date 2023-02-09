package rolluprelayer

import (
	"context"
	"time"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l2"
	"scroll-tech/bridge/sender"
	"scroll-tech/database"
	"scroll-tech/database/orm"
)

// L2RollupRelayer is struct to wrap l2.relayer to transmit rollup msg
type L2RollupRelayer struct {
	ctx      context.Context
	relayer  *l2.Layer2Relayer
	rollupCh <-chan *sender.Confirmation
	stopCh   chan struct{}
	db       orm.L2MessageOrm
}

// NewL2RollupRelayer creates a new instance of L2RollupRelayer
func NewL2RollupRelayer(ctx context.Context, cfg *config.RelayerConfig, db database.OrmFactory) (*L2RollupRelayer, error) {
	msgRelayer, err := l2.NewLayer2Relayer(ctx, db, cfg)
	if err != nil {
		return nil, err
	}
	return &L2RollupRelayer{
		ctx:      ctx,
		relayer:  msgRelayer,
		rollupCh: msgRelayer.GetRollupCh(),
		db:       db,
		stopCh:   make(chan struct{}),
	}, nil
}

// Start process
func (r *L2RollupRelayer) Start() {
	loop := func(ctx context.Context, f func()) {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				f()
			}
		}
	}
	go func() {
		ctx, cancel := context.WithCancel(r.ctx)
		// trigger by timer
		ticker := time.NewTicker(time.Second)
		defer cancel()

		for {
			select {
			case <-ticker.C:
				// To do: Refactoring this
				go loop(ctx, r.relayer.ProcessPendingBatches)
				go loop(ctx, r.relayer.ProcessCommittedBatches)
			case confirmation := <-r.rollupCh:
				r.relayer.HandleConfirmation(confirmation)
			case <-r.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop sends the stop signal to stop chan
func (r *L2RollupRelayer) Stop() {
	close(r.stopCh)
}
