package coordinator

import (
	"context"
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/scroll/common/message"
)

// RollerAPI for rollers inorder to register and submit proof
type RollerAPI interface {
	Register(ctx context.Context, authMsg *message.AuthMessage) (*rpc.Subscription, error)
	SubmitProof(proof *message.AuthZkProof) (bool, error)
}

// Register register api for roller
func (m *Manager) Register(ctx context.Context, authMsg *message.AuthMessage) (*rpc.Subscription, error) {
	// Verify register message.
	if ok, err := authMsg.Verify(); !ok {
		if err != nil {
			log.Error("failed to verify auth message", "error", err)
		}
		return nil, errors.New("signature verification failed")
	}

	pubkey, _ := authMsg.PublicKey()
	// create or get the roller message channel
	traceCh, err := m.register(pubkey, authMsg.Identity)
	if err != nil {
		return nil, err
	}

	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()
	go func() {
		defer func() {
			m.freeRoller(pubkey)
			log.Info("roller unregister", "name", authMsg.Name)
		}()

		for {
			select {
			case trace := <-traceCh:
				notifier.Notify(rpcSub.ID, trace)
			case <-rpcSub.Err():
				return
			case <-notifier.Closed():
				return
			}
		}
	}()
	log.Info("roller register", "name", authMsg.Name)

	return rpcSub, nil
}

// SubmitProof roller pull proof
func (m *Manager) SubmitProof(proof *message.AuthZkProof) (bool, error) {
	// Verify the signature
	if ok, err := proof.Verify(); !ok {
		if err != nil {
			log.Error("failed to verify proof message", "error", err)
		}
		return false, errors.New("auth signature verify fail")
	}

	pubkey, _ := proof.PublicKey()
	// Only allow registered pub-key.
	if !m.existID(pubkey, proof.ID) {
		return false, fmt.Errorf("the roller or session id doesn't exist, pubkey: %s, ID: %d", pubkey, proof.ID)
	}

	defer m.freeID(pubkey, proof.ID)

	err := m.handleZkProof(pubkey, proof.ProofMsg)
	if err != nil {
		return false, err
	}

	log.Info("Received zk proof", "proof id", proof.ID, "result", true)
	return true, nil
}
