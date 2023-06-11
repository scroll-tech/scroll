package roller

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math"
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

	"scroll-tech/roller/config"
	"scroll-tech/roller/prover"
	"scroll-tech/roller/store"
)

var (
	// retry connecting to coordinator
	retryWait = time.Second * 10
)

// Roller contains websocket conn to coordinator, Stack, unix-socket to ipc-prover.
type Roller struct {
	cfg         *config.Config
	client      *client.Client
	traceClient *ethclient.Client
	stack       *store.Stack
	prover      *prover.Prover
	taskChan    chan *message.TaskMsg
	sub         ethereum.Subscription

	isDisconnected int64
	isClosed       int64
	stopChan       chan struct{}

	priv *ecdsa.PrivateKey
}

// NewRoller new a Roller object.
func NewRoller(cfg *config.Config) (*Roller, error) {
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

	// Create prover instance
	log.Info("init prover")
	newProver, err := prover.NewProver(cfg.Prover)
	if err != nil {
		return nil, err
	}
	log.Info("init prover successfully!")

	rClient, err := client.Dial(cfg.CoordinatorURL)
	if err != nil {
		return nil, err
	}

	return &Roller{
		cfg:         cfg,
		client:      rClient,
		traceClient: traceClient,
		stack:       stackDb,
		prover:      newProver,
		sub:         nil,
		taskChan:    make(chan *message.TaskMsg, 10),
		stopChan:    make(chan struct{}),
		priv:        priv,
	}, nil
}

// Type returns roller type.
func (r *Roller) Type() message.ProveType {
	return r.cfg.Prover.ProveType
}

// PublicKey translate public key to hex and return.
func (r *Roller) PublicKey() string {
	return common.Bytes2Hex(crypto.CompressPubkey(&r.priv.PublicKey))
}

// Start runs Roller.
func (r *Roller) Start() {
	log.Info("start to register to coordinator")
	if err := r.Register(); err != nil {
		log.Crit("register to coordinator failed", "error", err)
	}
	log.Info("register to coordinator successfully!")

	go r.HandleCoordinator()
	go r.ProveLoop()
}

// Register registers Roller to the coordinator through Websocket.
func (r *Roller) Register() error {
	timestamp := time.Now().Unix()

	if timestamp < 0 || timestamp > math.MaxUint32 {
		panic("Expected current time to be between the years 1970 and 2106")
	}

	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:       r.cfg.RollerName,
			RollerType: r.Type(),
			Timestamp:  uint32(timestamp),
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
func (r *Roller) HandleCoordinator() {
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

func (r *Roller) mustRetryCoordinator() {
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
func (r *Roller) ProveLoop() {
	for {
		select {
		case <-r.stopChan:
			return
		default:
			if err := r.prove(); err != nil {
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

func (r *Roller) prove() error {
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

		log.Info("start to prove block", "task-id", task.Task.ID)

		var traces []*types.BlockTrace
		traces, err = r.getSortedTracesByHashes(task.Task.BlockHashes)
		if err != nil {
			return err
		}
		// If FFI panic during Prove, the roller will restart and re-enter prove() function,
		// the proof will not be submitted.
		var proof *message.AggProof
		proof, err = r.prover.Prove(task.Task.ID, traces)
		if err != nil {
			proofMsg = &message.ProofDetail{
				Status: message.StatusProofError,
				Error:  err.Error(),
				ID:     task.Task.ID,
				Type:   task.Task.Type,
				Proof:  &message.AggProof{},
			}
			log.Error("prove block failed!", "task-id", task.Task.ID)
		} else {
			proofMsg = &message.ProofDetail{
				Status: message.StatusOk,
				ID:     task.Task.ID,
				Type:   task.Task.Type,
				Proof:  proof,
			}
			log.Info("prove block successfully!", "task-id", task.Task.ID)
		}
	} else {
		// when the roller has more than 3 times panic,
		// it will omit to prove the task, submit StatusProofError and then Delete the task.
		proofMsg = &message.ProofDetail{
			Status: message.StatusProofError,
			Error:  "zk proving panic",
			ID:     task.Task.ID,
			Type:   task.Task.Type,
			Proof:  &message.AggProof{},
		}
	}

	defer func() {
		err = r.stack.Delete(task.Task.ID)
		if err != nil {
			log.Error("roller stack pop failed!", "err", err)
		}
	}()

	r.signAndSubmitProof(proofMsg)
	return nil
}

func (r *Roller) signAndSubmitProof(msg *message.ProofDetail) {
	authZkProof := &message.ProofMsg{ProofDetail: msg}
	if err := authZkProof.Sign(r.priv); err != nil {
		log.Error("sign proof error", "err", err)
		return
	}

	// Retry SubmitProof several times.
	for i := 0; i < 3; i++ {
		// When the roller is disconnected from the coordinator,
		// wait until the roller reconnects to the coordinator.
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

func (r *Roller) getSortedTracesByHashes(blockHashes []common.Hash) ([]*types.BlockTrace, error) {
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
func (r *Roller) Stop() {
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
