package l2

import (
	"context"
	"sync"
	"time"

	// not sure if this will make problems when relay with l1geth

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
)

// Layer2Relayer is responsible for
//  1. Committing and finalizing L2 blocks on L1
//  2. Relaying messages from L2 to L1
//
// Actions are triggered by new head from layer 1 geth node.
// @todo It's better to be triggered by watcher.
type Layer2Relayer struct {
	ctx context.Context

	db  database.OrmFactory
	cfg *config.RelayerConfig

	messageSender *sender.Sender
	messageCh     <-chan *sender.Confirmation

	rollupSender *sender.Sender
	rollupCh     <-chan *sender.Confirmation

	// A list of processing message.
	// key(string): confirmation ID, value(string): layer2 hash.
	processingMessage sync.Map

	// A list of processing batch commitment.
	// key(string): confirmation ID, value(string): batch id.
	processingCommitment sync.Map

	// A list of processing batch finalization.
	// key(string): confirmation ID, value(string): batch id.
	processingFinalization sync.Map

	stopCh chan struct{}
}

// NewLayer2Relayer will return a new instance of Layer2RelayerClient
func NewLayer2Relayer(ctx context.Context, db database.OrmFactory, cfg *config.RelayerConfig) (*Layer2Relayer, error) {
	// @todo use different sender for relayer, block commit and proof finalize
	messageSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.MessageSenderPrivateKeys)
	if err != nil {
		log.Error("Failed to create messenger sender", "err", err)
		return nil, err
	}

	rollupSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.RollupSenderPrivateKeys)
	if err != nil {
		log.Error("Failed to create rollup sender", "err", err)
		return nil, err
	}

	layer2 := &Layer2Relayer{
		ctx:                    ctx,
		db:                     db,
		messageSender:          messageSender,
		messageCh:              messageSender.ConfirmChan(),
		rollupSender:           rollupSender,
		rollupCh:               rollupSender.ConfirmChan(),
		cfg:                    cfg,
		processingMessage:      sync.Map{},
		processingCommitment:   sync.Map{},
		processingFinalization: sync.Map{},
		stopCh:                 make(chan struct{}),
	}

	if err = layer2.messageInit(); err != nil {
		log.Error("failed to init layer2 submitted tx", "err", err)
		return nil, err
	}

	if err = layer2.commitInit(); err != nil {
		log.Error("failed to init layer2 committed tx", "err", err)
		return nil, err
	}

	if err = layer2.finalizeInit(); err != nil {
		log.Error("failed to init layer2 finalized tx", "err", err)
		return nil, err
	}

	return layer2, nil
}

// Start the relayer process
func (r *Layer2Relayer) Start() {
	go func() {
		// trigger by timer
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				var wg = sync.WaitGroup{}
				wg.Add(3)
				go r.ProcessSavedEvents(&wg)
				go r.ProcessPendingBatches(&wg)
				go r.ProcessCommittedBatches(&wg)
				wg.Wait()
			case confirmation := <-r.messageCh:
				r.handleConfirmation(confirmation)
			case confirmation := <-r.rollupCh:
				r.handleConfirmation(confirmation)
			case <-r.stopCh:
				return
			}
		}
	}()
}

// Stop the relayer module, for a graceful shutdown.
func (r *Layer2Relayer) Stop() {
	close(r.stopCh)
}

func (r *Layer2Relayer) handleConfirmation(confirmation *sender.Confirmation) {
	if !confirmation.IsSuccessful {
		log.Warn("transaction confirmed but failed in layer1", "confirmation", confirmation)
		return
	}

	transactionType := "Unknown"
	// check whether it is message relay transaction
	if msgHash, ok := r.processingMessage.Load(confirmation.ID); ok {
		transactionType = "MessageRelay"
		// @todo handle db error
		err := r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, msgHash.(string), orm.MsgConfirmed, confirmation.TxHash.String())
		if err != nil {
			log.Warn("UpdateLayer2StatusAndLayer1Hash failed", "msgHash", msgHash.(string), "err", err)
		}
		r.processingMessage.Delete(confirmation.ID)
	}

	// check whether it is block commitment transaction
	if batchID, ok := r.processingCommitment.Load(confirmation.ID); ok {
		transactionType = "BatchCommitment"
		// @todo handle db error
		err := r.db.UpdateCommitTxHashAndRollupStatus(r.ctx, batchID.(string), confirmation.TxHash.String(), orm.RollupCommitted)
		if err != nil {
			log.Warn("UpdateCommitTxHashAndRollupStatus failed", "batch_id", batchID.(string), "err", err)
		}
		r.processingCommitment.Delete(confirmation.ID)
	}

	// check whether it is proof finalization transaction
	if batchID, ok := r.processingFinalization.Load(confirmation.ID); ok {
		transactionType = "ProofFinalization"
		// @todo handle db error
		err := r.db.UpdateFinalizeTxHashAndRollupStatus(r.ctx, batchID.(string), confirmation.TxHash.String(), orm.RollupFinalized)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "batch_id", batchID.(string), "err", err)
		}
		r.processingFinalization.Delete(confirmation.ID)
	}
	log.Info("transaction confirmed in layer1", "type", transactionType, "confirmation", confirmation)
}
