package messagerelayer

import (
	"context"
	"time"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"
	"scroll-tech/bridge/l2"
	"scroll-tech/bridge/sender"
)

// L1MsgRelayer wraps l1 relayer for message-relayer bin
type L1MsgRelayer struct {
	ctx    context.Context
	cancel context.CancelFunc

	relayer   *l1.Layer1Relayer
	confirmCh <-chan *sender.Confirmation
	db        orm.L1MessageOrm
}

// L2MsgRelayer wraps l2 relayer for message-relayer bin
type L2MsgRelayer struct {
	ctx    context.Context
	cancel context.CancelFunc

	relayer      *l2.Layer2Relayer
	msgConfirmCh <-chan *sender.Confirmation
	db           orm.L2MessageOrm
}

// NewL2MsgRelayer creates a new instance of L2MsgRelayer
func NewL2MsgRelayer(ctx context.Context, db database.OrmFactory, cfg *config.RelayerConfig) (*L2MsgRelayer, error) {
	msgRelayer, err := l2.NewLayer2Relayer(ctx, db, cfg)
	if err != nil {
		return nil, err
	}
	subCtx, cancel := context.WithCancel(ctx)
	return &L2MsgRelayer{
		ctx:          subCtx,
		cancel:       cancel,
		relayer:      msgRelayer,
		msgConfirmCh: msgRelayer.GetMsgConfirmCh(),
		db:           db,
	}, nil
}

// NewL1MsgRelayer creates a new instance of L1MsgRelayer
func NewL1MsgRelayer(ctx context.Context, l1ConfirmNum int64, db orm.L1MessageOrm, cfg *config.RelayerConfig) (*L1MsgRelayer, error) {
	msgRelayer, err := l1.NewLayer1Relayer(ctx, db, cfg)
	if err != nil {
		return nil, err
	}
	subCtx, cancel := context.WithCancel(ctx)
	return &L1MsgRelayer{
		ctx:       subCtx,
		cancel:    cancel,
		relayer:   msgRelayer,
		confirmCh: msgRelayer.GetConfirmCh(),
		db:        db,
	}, nil
}

// Start runs go routine to process saved events on L1
func (l1r *L1MsgRelayer) Start() {
	go func() {
		// trigger by timer
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				l1r.relayer.ProcessSavedEvents(l1r.ctx)
			case cfm := <-l1r.relayer.GetConfirmCh():
				if !cfm.IsSuccessful {
					log.Warn("transaction confirmed but failed in layer2", "confirmation", cfm)
				} else {
					// @todo handle db error
					err := l1r.db.UpdateLayer1StatusAndLayer2Hash(l1r.ctx, cfm.ID, orm.MsgConfirmed, cfm.TxHash.String())
					if err != nil {
						log.Warn("UpdateLayer1StatusAndLayer2Hash failed", "err", err)
					}
					log.Info("transaction confirmed in layer2", "confirmation", cfm)
				}
			}
		}
	}()
}

// Stop sends signal to stop chan
func (l1r *L1MsgRelayer) Stop() {
	if l1r.cancel != nil {
		l1r.cancel()
		l1r.cancel = nil
	}
}

// Start runs go routine to process saved events on L2
func (l2r *L2MsgRelayer) Start() {
	go func() {
		// trigger by timer
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				l2r.relayer.ProcessSavedEvents(l2r.ctx)
			case confirmation := <-l2r.relayer.GetMsgConfirmCh():
				l2r.relayer.HandleConfirmation(confirmation)
			case <-l2r.ctx.Done():
				return
			}
		}
	}()
}

// Stop sends signal to stop chan
func (l2r *L2MsgRelayer) Stop() {
	if l2r.cancel != nil {
		l2r.cancel()
		l2r.cancel = nil
	}
}
