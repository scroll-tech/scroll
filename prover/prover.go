package prover

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/coordinator/client"

	"scroll-tech/prover/config"
	"scroll-tech/prover/core"
	"scroll-tech/prover/store"
)

var (
	// retry connecting to coordinator
	retryWait = time.Second * 10
)

// Prover contains websocket conn to coordinator, and task stack.
type Prover struct {
	cfg         *config.Config
	client      *client.Client
	traceClient *ethclient.Client
	stack       *store.Stack
	proverCore  *core.ProverCore
	taskChan    chan *message.TaskMsg
	sub         ethereum.Subscription

	isDisconnected int64
	isClosed       int64
	stopChan       chan struct{}

	priv *ecdsa.PrivateKey
}

// NewProver new a Prover object.
func NewProver(cfg *config.Config) (*Prover, error) {
	// load or create wallet
	priv, err := utils.LoadOrCreateKey(cfg.KeystorePath, cfg.KeystorePassword)
	if err != nil {
		return nil, err
	}

	// Get stack db handler
	stackDb, err := store.NewStack(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	// Collect geth node.
	traceClient, err := ethclient.Dial(cfg.TraceEndpoint)
	if err != nil {
		return nil, err
	}

	// Create prover_core instance
	log.Info("init prover_core")
	newProverCore, err := core.NewProverCore(cfg.Core)
	if err != nil {
		return nil, err
	}
	log.Info("init prover_core successfully!")

	rClient, err := client.Dial(cfg.CoordinatorURL)
	if err != nil {
		return nil, err
	}

	return &Prover{
		cfg:         cfg,
		client:      rClient,
		traceClient: traceClient,
		stack:       stackDb,
		proverCore:  newProverCore,
		sub:         nil,
		taskChan:    make(chan *message.TaskMsg, 10),
		stopChan:    make(chan struct{}),
		priv:        priv,
	}, nil
}

// Type returns prover type.
func (r *Prover) Type() message.ProofType {
	return r.cfg.Core.ProofType
}

// PublicKey translate public key to hex and return.
func (r *Prover) PublicKey() string {
	return common.Bytes2Hex(crypto.CompressPubkey(&r.priv.PublicKey))
}

// Start runs Prover.
func (r *Prover) Start() {
	log.Info("start to register to coordinator")
	if err := r.Register(); err != nil {
		log.Crit("register to coordinator failed", "error", err)
	}
	log.Info("register to coordinator successfully!")

	go r.HandleCoordinator()
	go r.ProveLoop()
}

// Register registers Prover to the coordinator through Websocket.
func (r *Prover) Register() error {
	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:       r.cfg.ProverName,
			ProverType: r.Type(),
			Version:    version.Version,
		},
	}
	// Sign request token message
	if err := authMsg.SignWithKey(r.priv); err != nil {
		return fmt.Errorf("sign request token message failed %v", err)
	}

	token, err := r.client.RequestToken(context.Background(), authMsg)
	if err != nil {
		return fmt.Errorf("request token failed %v", err)
	}
	authMsg.Identity.Token = token

	// Sign auth message
	if err = authMsg.SignWithKey(r.priv); err != nil {
		return fmt.Errorf("sign auth message failed %v", err)
	}

	sub, err := r.client.RegisterAndSubscribe(context.Background(), r.taskChan, authMsg)
	r.sub = sub
	return err
}

// HandleCoordinator accepts block-traces from coordinator through the Websocket and store it into Stack.
func (r *Prover) HandleCoordinator() {
	for {
		select {
		case <-r.stopChan:
			return
		case task := <-r.taskChan:
			log.Info("Accept BlockTrace from Scroll", "ID", task.ID)
			err := r.stack.Push(&store.ProvingTask{Task: task, Times: 0})
			if err != nil {
				panic(fmt.Sprintf("could not push task(%s) into stack: %v", task.ID, err))
			}
		case err := <-r.sub.Err():
			r.sub.Unsubscribe()
			log.Error("Subscribe task with scroll failed", "error", err)
			if atomic.LoadInt64(&r.isClosed) == 0 {
				r.mustRetryCoordinator()
			}
		}
	}
}

func (r *Prover) mustRetryCoordinator() {
	atomic.StoreInt64(&r.isDisconnected, 1)
	defer atomic.StoreInt64(&r.isDisconnected, 0)
	for {
		log.Info("retry to connect to coordinator...")
		err := r.Register()
		if err != nil {
			log.Error("register to coordinator failed", "error", err)
			time.Sleep(retryWait)
		} else {
			log.Info("re-register to coordinator successfully!")
			break
		}
	}

}

// ProveLoop keep popping the block-traces from Stack and sends it to rust-prover for loop.
func (r *Prover) ProveLoop() {
	for {
		select {
		case <-r.stopChan:
			return
		default:
			if err := r.proveAndSubmit(); err != nil {
				if errors.Is(err, store.ErrEmpty) {
					log.Debug("get empty trace", "error", err)
					time.Sleep(time.Second * 3)
					continue
				}
				log.Error("prove failed", "error", err)
			}
		}
	}
}

func (r *Prover) proveAndSubmit() error {
	task, err := r.stack.Peek()
	if err != nil {
		return err
	}

	var proofMsg *message.ProofDetail
	if task.Times <= 2 {
		// If panic times <= 2, try to proof the task.
		if err = r.stack.UpdateTimes(task, task.Times+1); err != nil {
			return err
		}

		log.Info("start to prove task", "task-type", task.Task.Type, "task-id", task.Task.ID)
		proofMsg = r.prove(task)
	} else {
		// when the prover has more than 3 times panic,
		// it will omit to prove the task, submit StatusProofError and then Delete the task.
		proofMsg = &message.ProofDetail{
			Status: message.StatusProofError,
			Error:  "zk proving panic",
			ID:     task.Task.ID,
			Type:   task.Task.Type,
		}
	}

	defer func() {
		err = r.stack.Delete(task.Task.ID)
		if err != nil {
			log.Error("prover stack pop failed!", "err", err)
		}
	}()

	r.signAndSubmitProof(proofMsg)
	return nil
}

func (r *Prover) prove(task *store.ProvingTask) (detail *message.ProofDetail) {
	detail = &message.ProofDetail{
		ID:     task.Task.ID,
		Type:   task.Task.Type,
		Status: message.StatusOk,
	}

	if r.Type() == message.ProofTypeChunk {
		proof, err := r.proveChunk(task)
		if err != nil {
			log.Error("prove chunk failed!", "task-id", task.Task.ID)
			detail.Status = message.StatusProofError
			detail.Error = err.Error()
			return
		}
		detail.ChunkProof = proof
		log.Info("prove chunk successfully!", "task-id", task.Task.ID)
		return
	} else{
		proof, err := r.proveBatch(task)
		if err != nil {
			log.Error("prove batch failed!", "task-id", task.Task.ID)
			detail.Status = message.StatusProofError
			detail.Error = err.Error()
			return
		}
		detail.BatchProof = proof
		log.Info("prove batch successfully!", "task-id", task.Task.ID)
		return
	}
}

func (r *Prover) proveChunk(task *store.ProvingTask) (*message.ChunkProof, error) {
	traces, err := r.getSortedTracesByHashes(task.Task.BlockHashes)
	if err != nil {
		log.Error("get traces failed!", "task-id", task.Task.ID, "err", err)
		return nil, errors.New("get traces from eth node failed")
	}
	return r.proverCore.ProveChunk(task.Task.ID, traces)
}

func (r *Prover) proveBatch(task *store.ProvingTask) (*message.BatchProof, error) {
	return r.proverCore.ProveBatch(task.Task.ID, task.Task.ChunkHashes, task.Task.SubProofs)
}

func (r *Prover) signAndSubmitProof(msg *message.ProofDetail) {
	authZkProof := &message.ProofMsg{ProofDetail: msg}
	if err := authZkProof.Sign(r.priv); err != nil {
		log.Error("sign proof error", "err", err)
		return
	}

	// Retry SubmitProof several times.
	for i := 0; i < 3; i++ {
		// When the prover is disconnected from the coordinator,
		// wait until the prover reconnects to the coordinator.
		for atomic.LoadInt64(&r.isDisconnected) == 1 {
			time.Sleep(retryWait)
		}
		serr := r.client.SubmitProof(context.Background(), authZkProof)
		if serr == nil {
			return
		}
		log.Error("submit proof to coordinator error", "task ID", msg.ID, "error", serr)
	}
}

func (r *Prover) getSortedTracesByHashes(blockHashes []common.Hash) ([]*types.BlockTrace, error) {
	var traces []*types.BlockTrace
	for _, blockHash := range blockHashes {
		trace, err := r.traceClient.GetBlockTraceByHash(context.Background(), blockHash)
		if err != nil {
			return nil, err
		}
		traces = append(traces, trace)
	}
	// Sort BlockTraces by header number.
	// TODO: we should check that the number range here is continuous.
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].Header.Number.Int64() < traces[j].Header.Number.Int64()
	})
	return traces, nil
}

// Stop closes the websocket connection.
func (r *Prover) Stop() {
	if atomic.LoadInt64(&r.isClosed) == 1 {
		return
	}
	atomic.StoreInt64(&r.isClosed, 1)

	close(r.stopChan)
	// Close scroll's ws
	r.sub.Unsubscribe()
	// Close db
	if err := r.stack.Close(); err != nil {
		log.Error("failed to close bbolt db", "error", err)
	}
}
