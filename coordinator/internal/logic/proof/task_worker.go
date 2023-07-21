package proof

import (
	"context"
	"fmt"

	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/logic/rollermanager"
)

var coordinatorRollersDisconnectsTotalCounter = gethMetrics.NewRegisteredCounter("coordinator/rollers/disconnects/total", metrics.ScrollRegistry)

// TaskWorker held the roller task connection
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

	// create or get the roller message channel
	taskCh, err := rollermanager.Manager.Register(pubKey, identity)
	if err != nil {
		return &rpc.Subscription{}, err
	}

	rpcSub := notifier.CreateSubscription()

	go t.worker(rpcSub, notifier, pubKey, identity, taskCh)

	log.Info("roller register", "name", identity.Name, "pubKey", pubKey, "version", identity.Version)

	return rpcSub, nil
}

func (t *TaskWorker) worker(rpcSub *rpc.Subscription, notifier *rpc.Notifier, pubKey string, identity *message.Identity, taskCh <-chan *message.TaskMsg) {
	defer func() {
		if err := recover(); err != nil {
			log.Error("task worker subId:%d panic for:%v", err)
		}

		rollermanager.Manager.FreeRoller(pubKey)
		log.Info("roller unregister", "name", identity.Name, "pubKey", pubKey)
	}()

	for {
		select {
		case task := <-taskCh:
			notifier.Notify(rpcSub.ID, task) //nolint
		case err := <-rpcSub.Err():
			coordinatorRollersDisconnectsTotalCounter.Inc(1)
			log.Warn("client stopped the ws connection", "name", identity.Name, "pubkey", pubKey, "err", err)
			return
		case <-notifier.Closed():
			return
		}
	}
}
