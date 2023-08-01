package prover

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"
	"scroll-tech/prover/client"
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
	ctx               context.Context
	cfg               *config.Config
	coordinatorClient *client.CoordinatorClient
	traceClient       *ethclient.Client
	stack             *store.Stack
	proverCore        *core.ProverCore

	isClosed int64
	stopChan chan struct{}

	priv *ecdsa.PrivateKey
}

// NewProver new a Prover object.
func NewProver(ctx context.Context, cfg *config.Config) (*Prover, error) {
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
	traceClient, err := ethclient.DialContext(ctx, cfg.TraceEndpoint)
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

	coordinatorClient, err := client.NewCoordinatorClient(cfg.Coordinator)
	if err != nil {
		return nil, err
	}

	return &Prover{
		ctx:               ctx,
		cfg:               cfg,
		coordinatorClient: coordinatorClient,
		traceClient:       traceClient,
		stack:             stackDb,
		proverCore:        newProverCore,
		stopChan:          make(chan struct{}),
		priv:              priv,
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
	log.Info("start to login to coordinator")
	if _, err := r.coordinatorClient.Login(r.ctx, &client.ProverLoginRequest{
		PublicKey:  r.PublicKey(),
		ProverName: r.cfg.ProverName,
	}); err != nil {
		log.Crit("login to coordinator failed", "error", err)
	}
	log.Info("login to coordinator successfully!")

	go r.ProveLoop()
}

// ProveLoop keep popping the block-traces from Stack and sends it to rust-prover for loop.
func (r *Prover) ProveLoop() {
	for {
		select {
		case <-r.stopChan:
			return
		default:
			if err := r.prove(); err != nil {
				log.Error("prove failed", "error", err)
			}
		}
	}
}

func (r *Prover) prove() error {
	task, err := r.stack.Peek()
	if err != nil {
		if err != store.ErrEmpty {
			return err
		}
		// fetch new proving task.
		task, err = r.fetchTaskFromServer()
		if err != nil {
			time.Sleep(retryWait)
			return err
		}
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
			proofMsg = &message.ProofDetail{
				Status:     message.StatusProofError,
				Error:      "get traces failed",
				ID:         task.Task.ID,
				Type:       task.Task.Type,
				ChunkProof: nil,
			}
			log.Error("get traces failed!", "task-id", task.Task.ID, "err", err)
		} else {
			// If FFI panic during Prove, the prover will restart and re-enter prove() function,
			// the proof will not be submitted.
			var proof *message.ChunkProof
			proof, err = r.proverCore.Prove(task.Task.ID, traces)
			if err != nil {
				proofMsg = &message.ProofDetail{
					Status:     message.StatusProofError,
					Error:      err.Error(),
					ID:         task.Task.ID,
					Type:       task.Task.Type,
					ChunkProof: nil,
				}
				log.Error("prove block failed!", "task-id", task.Task.ID)
			} else {
				proofMsg = &message.ProofDetail{
					Status:     message.StatusOk,
					ID:         task.Task.ID,
					Type:       task.Task.Type,
					ChunkProof: proof,
				}
				log.Info("prove block successfully!", "task-id", task.Task.ID)
			}
		}
	} else {
		// when the prover has more than 3 times panic,
		// it will omit to prove the task, submit StatusProofError and then Delete the task.
		proofMsg = &message.ProofDetail{
			Status:     message.StatusProofError,
			Error:      "zk proving panic",
			ID:         task.Task.ID,
			Type:       task.Task.Type,
			ChunkProof: nil,
		}
	}

	defer func() {
		err = r.stack.Delete(task.Task.ID)
		if err != nil {
			log.Error("prover stack pop failed!", "err", err)
		}
	}()

	return r.signAndSubmitProof(proofMsg)
}

// fetchTaskFromServer fetches a new task from the server
func (r *Prover) fetchTaskFromServer() (*store.ProvingTask, error) {
	// Prepare the request
	req := &client.ProverTasksRequest{
		ProverVersion: version.Version,
		ProverHeight:  0, // TODO(colinlyguo): adjust prover height.
		ProofType:     int(r.Type()),
	}

	// Send the request
	resp, err := r.coordinatorClient.ProverTasks(r.ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return nil, fmt.Errorf("no tasks available")
	}

	// Convert the response task to a ProvingTask
	provingTask := &store.ProvingTask{
		Task: &message.TaskMsg{
			ID:   resp.Data.TaskID,
			Type: resp.Data.ProofType,
		},
		Times: 0,
	}

	switch resp.Data.ProofType {
	case message.ProofTypeChunk:
		var blockHashes []common.Hash
		err := json.Unmarshal([]byte(resp.Data.ProofData), &blockHashes)
		if err != nil {
			return nil, err
		}
		provingTask.Task.BlockHashes = blockHashes
	case message.ProofTypeBatch:
		var subProofs []*message.ChunkProof
		err := json.Unmarshal([]byte(resp.Data.ProofData), &subProofs)
		if err != nil {
			return nil, err
		}
		provingTask.Task.SubProofs = subProofs
	default:
		return nil, fmt.Errorf("unknown proof type: %d", resp.Data.ProofType)
	}

	return provingTask, nil
}

func (r *Prover) signAndSubmitProof(msg *message.ProofDetail) error {
	authZkProof := &message.ProofMsg{ProofDetail: msg}
	if err := authZkProof.Sign(r.priv); err != nil {
		return fmt.Errorf("error signing proof: %v", err)
	}

	// Marshal the ChunkProof and BatchProof into a single JSON
	proofs := map[string]interface{}{
		"chunk_proof": authZkProof.ChunkProof,
		"batch_proof": authZkProof.BatchProof,
	}

	proofJson, err := json.Marshal(proofs)
	if err != nil {
		return fmt.Errorf("error marshaling proofs into JSON: %v", err)
	}

	// Prepare the submit request
	req := &client.SubmitProofRequest{
		TaskID:    authZkProof.ProofDetail.ID,
		Status:    int(authZkProof.ProofDetail.Status),
		Error:     msg.Error,
		ProofType: int(authZkProof.ProofDetail.Type),
		Signature: authZkProof.Signature,
		Proof:     string(proofJson),
	}

	// Send the submit request
	resp, err := r.coordinatorClient.SubmitProof(r.ctx, req)
	if err != nil {
		return fmt.Errorf("error submitting proof: %v", err)
	}

	if resp.ErrCode != 200 {
		return fmt.Errorf("submit proof error, error code: %v, error message: %v", resp.ErrCode, resp.ErrMsg)
	}

	log.Debug("proof submitted successfully", "task-id", msg.ID)
	return nil
}

func (r *Prover) getSortedTracesByHashes(blockHashes []common.Hash) ([]*types.BlockTrace, error) {
	var traces []*types.BlockTrace
	for _, blockHash := range blockHashes {
		trace, err := r.traceClient.GetBlockTraceByHash(r.ctx, blockHash)
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
	// Close db
	if err := r.stack.Close(); err != nil {
		log.Error("failed to close bbolt db", "error", err)
	}
}
