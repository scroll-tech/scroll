package proof

import (
	"context"
	"fmt"

	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/logic/provermanager"
)

var coordinatorProversDisconnectsTotalCounter = gethMetrics.NewRegisteredCounter("coordinator/provers/disconnects/total", metrics.ScrollRegistry)

// TaskWorker held the prover task connection
type TaskWorker struct{}

// NewTaskWorker create a task worker
func NewTaskWorker() *TaskWorker {
	return &TaskWorker{}
}

// AllocTaskWorker alloc a task worker goroutine
func (t *TaskWorker) AllocTaskWorker(ctx context.Context, authMsg *message.AuthMsg) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	pubKey, err := authMsg.PublicKey()
	if err != nil {
		return &rpc.Subscription{}, fmt.Errorf("AllocTaskWorker auth msg public key error:%w", err)
	}

	identity := authMsg.Identity

	// create or get the prover message channel
	taskCh, err := provermanager.Manager.Register(ctx, pubKey, identity)
	if err != nil {
		return &rpc.Subscription{}, err
	}

	rpcSub := notifier.CreateSubscription()

	go t.worker(rpcSub, notifier, pubKey, identity, taskCh)

	log.Info("prover register", "name", identity.Name, "pubKey", pubKey, "version", identity.Version)

	return rpcSub, nil
}

// TODO worker add metrics
func (t *TaskWorker) worker(rpcSub *rpc.Subscription, notifier *rpc.Notifier, pubKey string, identity *message.Identity, taskCh <-chan *message.TaskMsg) {
	defer func() {
		if err := recover(); err != nil {
			log.Error("task worker subId:%d panic for:%v", err)
		}

		provermanager.Manager.FreeProver(pubKey)
		log.Info("prover unregister", "name", identity.Name, "pubKey", pubKey)
	}()

	for {
		select {
		case task := <-taskCh:
			notifier.Notify(rpcSub.ID, task) //nolint
		case err := <-rpcSub.Err():
			coordinatorProversDisconnectsTotalCounter.Inc(1)
			log.Warn("client stopped the ws connection", "name", identity.Name, "pubkey", pubKey, "err", err)
			return
		case <-notifier.Closed():
			return
		}
	}
}
