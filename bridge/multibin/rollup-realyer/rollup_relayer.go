package rolluprealyer

import (
	"context"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l2"
	"scroll-tech/bridge/sender"
	"scroll-tech/database"
	"scroll-tech/database/orm"
)

// L2RollupRelayer is struct to wrap l2.relayer to transmit rollup msg
type L2RollupRelayer struct {
	ctx          context.Context
	relayer      *l2.Layer2Relayer
	msgConfirmCh <-chan *sender.Confirmation
	stop         chan struct{}
	db           orm.L2MessageOrm
}

// NewL2RollupRelayer creates a new instance of L2RollupRelayer
func NewL2RollupRelayer(ctx context.Context, client *ethclient.Client, cfg *config.RelayerConfig, db database.OrmFactory) (*L2RollupRelayer, error) {
	msgRelayer, err := l2.NewLayer2Relayer(ctx, db, cfg)
	if err != nil {
		return nil, err
	}
	return &L2RollupRelayer{
		ctx:          ctx,
		relayer:      msgRelayer,
		msgConfirmCh: msgRelayer.GetMsgConfirmCh(),
		db:           db,
		stop:         make(chan struct{}),
	}, nil
}

// Start runs go routine to fetch contract events on L2
func (r *L2RollupRelayer) Start() {
	go func() {
		// trigger by timer
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				var wg = sync.WaitGroup{}
				wg.Add(2)
				go r.relayer.ProcessPendingBatches(&wg)
				go r.relayer.ProcessCommittedBatches(&wg)
				wg.Wait()
			case confirmation := <-r.msgConfirmCh:
				r.relayer.HandleConfirmation(confirmation)
			case <-r.stop:
				return
			}
		}
	}()
}

// Stop sends the stop signal to stop chan
func (l2w *L2EventWatcher) Stop() {
	l2w.stop <- struct{}{}
}
