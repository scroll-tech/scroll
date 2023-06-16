package coordinator

import (
	"context"
	"errors"
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/scroll-tech/go-ethereum/log"
	geth_metrics "github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types/message"
)

var (
	coordinatorRollersDisconnectsTotalCounter = geth_metrics.NewRegisteredCounter("coordinator/rollers/disconnects/total", metrics.ScrollRegistry)
)

// RollerAPI for rollers inorder to register and submit proof
type RollerAPI interface {
	RequestToken(authMsg *message.AuthMsg) (string, error)
	Register(ctx context.Context, authMsg *message.AuthMsg) (*rpc.Subscription, error)
	SubmitProof(proof *message.ProofMsg) error
}

// CoordinatorAPI for Coordinator in order to manage process.
type CoordinatorAPI interface {
	StartSend()
	PauseSend()
}

// RequestToken generates and sends back register token for roller
func (m *Manager) RequestToken(authMsg *message.AuthMsg) (string, error) {
	if ok, err := authMsg.Verify(); !ok {
		if err != nil {
			log.Error("failed to verify auth message", "error", err)
		}
		return "", errors.New("signature verification failed")
	}
	pubkey, _ := authMsg.PublicKey()
	if token, ok := m.tokenCache.Get(pubkey); ok {
		return token.(string), nil
	}
	token, err := message.GenerateToken()
	if err != nil {
		return "", errors.New("token generation failed")
	}
	m.tokenCache.Set(pubkey, token, cache.DefaultExpiration)
	return token, nil
}

// Register register api for roller
func (m *Manager) Register(ctx context.Context, authMsg *message.AuthMsg) (*rpc.Subscription, error) {
	// Verify register message.
	if ok, err := authMsg.Verify(); !ok {
		if err != nil {
			log.Error("failed to verify auth message", "error", err)
		}
		return nil, errors.New("signature verification failed")
	}
	pubkey, _ := authMsg.PublicKey()

	// Lock here to avoid malicious roller message replay before cleanup of token
	m.registerMu.Lock()
	if ok, err := m.VerifyToken(authMsg); !ok {
		m.registerMu.Unlock()
		return nil, err
	}
	// roller successfully registered, remove token associated with this roller
	m.tokenCache.Delete(pubkey)
	m.registerMu.Unlock()

	// create or get the roller message channel
	taskCh, err := m.register(pubkey, authMsg.Identity)
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
			log.Info("roller unregister", "name", authMsg.Identity.Name, "pubkey", pubkey)
		}()

		for {
			select {
			case task := <-taskCh:
				notifier.Notify(rpcSub.ID, task) //nolint
			case err := <-rpcSub.Err():
				coordinatorRollersDisconnectsTotalCounter.Inc(1)
				log.Warn("client stopped the ws connection", "name", authMsg.Identity.Name, "pubkey", pubkey, "err", err)
				return
			case <-notifier.Closed():
				return
			}
		}
	}()
	log.Info("roller register", "name", authMsg.Identity.Name, "pubkey", pubkey, "version", authMsg.Identity.Version)

	return rpcSub, nil
}

// SubmitProof roller pull proof
func (m *Manager) SubmitProof(proof *message.ProofMsg) error {
	// Verify the signature
	if ok, err := proof.Verify(); !ok {
		if err != nil {
			log.Error("failed to verify proof message", "error", err)
		}
		return errors.New("auth signature verify fail")
	}

	pubkey, _ := proof.PublicKey()
	// Only allow registered pub-key.
	if !m.existTaskIDForRoller(pubkey, proof.ID) {
		return fmt.Errorf("the roller or session id doesn't exist, pubkey: %s, ID: %s", pubkey, proof.ID)
	}

	m.updateMetricRollerProofsLastFinishedTimestampGauge(pubkey)

	err := m.handleZkProof(pubkey, proof.ProofDetail)
	if err != nil {
		return err
	}
	defer m.freeTaskIDForRoller(pubkey, proof.ID)

	return nil
}

// StartSend starts to send basic tasks.
func (m *Manager) StartSend() {
	m.StartSendTask()
}

// PauseSend pauses to send basic tasks.
func (m *Manager) PauseSend() {
	m.PauseSendTask()
}
