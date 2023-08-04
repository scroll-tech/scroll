package test

import (
	"context"
	"crypto/ecdsa"
	"sync"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types/message"

	client2 "scroll-tech/coordinator/client"
	"scroll-tech/coordinator/internal/logic/verifier"
)

type proofStatus uint32

const (
	verifiedSuccess proofStatus = iota
	verifiedFailed
	generatedFailed
)

type mockProver struct {
	proverName string
	privKey    *ecdsa.PrivateKey
	proofType  message.ProofType

	wsURL  string
	client *client2.Client

	taskCh    chan *message.TaskMsg
	taskCache sync.Map

	sub    ethereum.Subscription
	stopCh chan struct{}
}

func newMockProver(t *testing.T, proverName string, wsURL string, proofType message.ProofType) *mockProver {
	privKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	prover := &mockProver{
		proverName: proverName,
		privKey:    privKey,
		proofType:  proofType,
		wsURL:      wsURL,
		taskCh:     make(chan *message.TaskMsg, 4),
		stopCh:     make(chan struct{}),
	}
	prover.client, prover.sub, err = prover.connectToCoordinator()
	assert.NoError(t, err)

	return prover
}

// connectToCoordinator sets up a websocket client to connect to the prover manager.
func (r *mockProver) connectToCoordinator() (*client2.Client, ethereum.Subscription, error) {
	// Create connection.
	client, err := client2.Dial(r.wsURL)
	if err != nil {
		return nil, nil, err
	}

	// create a new ws connection
	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:       r.proverName,
			ProverType: r.proofType,
		},
	}
	_ = authMsg.SignWithKey(r.privKey)

	token, err := client.RequestToken(context.Background(), authMsg)
	if err != nil {
		return nil, nil, err
	}
	authMsg.Identity.Token = token
	_ = authMsg.SignWithKey(r.privKey)

	sub, err := client.RegisterAndSubscribe(context.Background(), r.taskCh, authMsg)
	if err != nil {
		return nil, nil, err
	}

	return client, sub, nil
}

func (r *mockProver) releaseTasks() {
	r.taskCache.Range(func(key, value any) bool {
		r.taskCh <- value.(*message.TaskMsg)
		r.taskCache.Delete(key)
		return true
	})
}

// Wait for the proof task, after receiving the proof task, prover submits proof after proofTime secs.
func (r *mockProver) waitTaskAndSendProof(t *testing.T, proofTime time.Duration, reconnect bool, proofStatus proofStatus) {
	// simulating the case that the prover first disconnects and then reconnects to the coordinator
	// the Subscription and its `Err()` channel will be closed, and the coordinator will `freeProver()`
	if reconnect {
		var err error
		r.client, r.sub, err = r.connectToCoordinator()
		if err != nil {
			t.Fatal(err)
			return
		}
	}

	// Release cached tasks.
	r.releaseTasks()

	go r.loop(t, r.client, proofTime, proofStatus)
}

func (r *mockProver) loop(t *testing.T, client *client2.Client, proofTime time.Duration, proofStatus proofStatus) {
	for {
		select {
		case task := <-r.taskCh:
			r.taskCache.Store(task.ID, task)
			// simulate proof time
			select {
			case <-time.After(proofTime):
			case <-r.stopCh:
				return
			}
			proof := &message.ProofMsg{
				ProofDetail: &message.ProofDetail{
					ID:         task.ID,
					Type:       r.proofType,
					Status:     message.StatusOk,
					ChunkProof: &message.ChunkProof{},
					BatchProof: &message.BatchProof{},
				},
			}
			if proofStatus == generatedFailed {
				proof.Status = message.StatusProofError
			} else if proofStatus == verifiedFailed {
				proof.ProofDetail.ChunkProof.Proof = []byte(verifier.InvalidTestProof)
				proof.ProofDetail.BatchProof.Proof = []byte(verifier.InvalidTestProof)
			}
			assert.NoError(t, proof.Sign(r.privKey))
			assert.NoError(t, client.SubmitProof(context.Background(), proof))
		case <-r.stopCh:
			return
		}
	}
}

func (r *mockProver) close() {
	close(r.stopCh)
	r.sub.Unsubscribe()
}
