package prover

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/prover/client"
	"scroll-tech/prover/config"
	"scroll-tech/prover/core"
	"scroll-tech/prover/store"
	putils "scroll-tech/prover/utils"

	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
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
	stack             *store.Stack
	l2GethClient      *ethclient.Client // only applicable for a chunk_prover
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

	var l2GethClient *ethclient.Client
	if cfg.Core.ProofType == message.ProofTypeChunk {
		if cfg.L2Geth == nil || cfg.L2Geth.Endpoint == "" {
			return nil, errors.New("Missing l2geth config for chunk prover")
		}
		// Connect l2geth node. Only applicable for a chunk_prover.
		l2GethClient, err = ethclient.DialContext(ctx, cfg.L2Geth.Endpoint)
		if err != nil {
			return nil, err
		}
		// Use gzip compression.
		l2GethClient.SetHeader("Accept-Encoding", "gzip")
	}

	// Create prover_core instance
	log.Info("init prover_core")
	newProverCore, err := core.NewProverCore(cfg.Core)
	if err != nil {
		return nil, err
	}
	log.Info("init prover_core successfully!")

	coordinatorClient, err := client.NewCoordinatorClient(cfg.Coordinator, cfg.ProverName, priv)
	if err != nil {
		return nil, err
	}

	return &Prover{
		ctx:               ctx,
		cfg:               cfg,
		coordinatorClient: coordinatorClient,
		l2GethClient:      l2GethClient,
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
	if err := r.coordinatorClient.Login(r.ctx); err != nil {
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
			if err := r.proveAndSubmit(); err != nil {
				log.Error("proveAndSubmit", "prover type", r.cfg.Core.ProofType, "error", err)
			}
		}
	}
}

func (r *Prover) proveAndSubmit() error {
	task, err := r.stack.Peek()
	if err != nil {
		if !errors.Is(err, store.ErrEmpty) {
			return fmt.Errorf("failed to peek from stack: %v", err)
		}
		// fetch new proving task.
		task, err = r.fetchTaskFromCoordinator()
		if err != nil {
			time.Sleep(retryWait)
			return fmt.Errorf("failed to fetch task from coordinator: %v", err)
		}

		// Push the new task into the stack
		if err = r.stack.Push(task); err != nil {
			return fmt.Errorf("failed to push task into stack: %v", err)
		}
	}

	var proofMsg *message.ProofDetail
	if task.Times <= 2 {
		// If tried times <= 2, try to proof the task.
		if err = r.stack.UpdateTimes(task, task.Times+1); err != nil {
			return fmt.Errorf("failed to update times on stack: %v", err)
		}

		log.Info("start to prove task", "task-type", task.Task.Type, "task-id", task.Task.ID)
		proofMsg, err = r.prove(task)
		if err != nil { // handling error from prove
			log.Error("failed to prove task", "task_type", task.Task.Type, "task-id", task.Task.ID, "err", err)
			return r.submitErr(task, message.ProofFailureNoPanic, err)
		}
		return r.submitProof(proofMsg, task.Task.UUID)
	}

	// if tried times >= 3, it's probably due to circuit proving panic
	log.Error("zk proving panic for task", "task-type", task.Task.Type, "task-id", task.Task.ID)
	return r.submitErr(task, message.ProofFailurePanic, errors.New("zk proving panic for task"))
}

// fetchTaskFromCoordinator fetches a new task from the server
func (r *Prover) fetchTaskFromCoordinator() (*store.ProvingTask, error) {
	// prepare the request
	req := &client.GetTaskRequest{
		TaskType: r.Type(),
		// we may not be able to get the vk at the first time, so we should pass vk to the coordinator every time we getTask
		// instead of passing vk when we login
		VK: r.proverCore.VK,
	}

	if req.TaskType == message.ProofTypeChunk {
		// get the latest confirmed block number
		latestBlockNumber, err := putils.GetLatestConfirmedBlockNumber(r.ctx, r.l2GethClient, r.cfg.L2Geth.Confirmations)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch latest confirmed block number: %v", err)
		}

		if latestBlockNumber == 0 {
			return nil, fmt.Errorf("omit to prove task of the genesis block, latestBlockNumber: %v", latestBlockNumber)
		}
		req.ProverHeight = latestBlockNumber
	}

	// send the request
	resp, err := r.coordinatorClient.GetTask(r.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get task, req: %v, err: %v", req, err)
	}

	// create a new TaskMsg
	taskMsg := message.TaskMsg{
		UUID: resp.Data.UUID,
		ID:   resp.Data.TaskID,
		Type: message.ProofType(resp.Data.TaskType),
	}

	// depending on the task type, unmarshal the task data into the appropriate field
	switch taskMsg.Type {
	case message.ProofTypeBatch:
		taskMsg.BatchTaskDetail = &message.BatchTaskDetail{}
		if err = json.Unmarshal([]byte(resp.Data.TaskData), taskMsg.BatchTaskDetail); err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch task detail: %v", err)
		}
	case message.ProofTypeChunk:
		taskMsg.ChunkTaskDetail = &message.ChunkTaskDetail{}
		if err = json.Unmarshal([]byte(resp.Data.TaskData), taskMsg.ChunkTaskDetail); err != nil {
			return nil, fmt.Errorf("failed to unmarshal chunk task detail: %v", err)
		}
	default:
		return nil, fmt.Errorf("unknown task type: %v", taskMsg.Type)
	}

	// convert the response task to a ProvingTask
	provingTask := &store.ProvingTask{
		Task:  &taskMsg,
		Times: 0,
	}

	// marshal the task to a json string for logging
	taskJSON, err := json.Marshal(provingTask)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task to json: %v", err)
	}

	log.Info("successfully fetched new task from coordinator", "resp", resp, "task", string(taskJSON))

	return provingTask, nil
}

// prove function tries to prove a task. It returns an error if the proof fails.
func (r *Prover) prove(task *store.ProvingTask) (*message.ProofDetail, error) {
	detail := &message.ProofDetail{
		ID:     task.Task.ID,
		Type:   task.Task.Type,
		Status: message.StatusOk,
	}

	switch r.Type() {
	case message.ProofTypeChunk:
		proof, err := r.proveChunk(task)
		if err != nil {
			detail.Status = message.StatusProofError
			detail.Error = err.Error()
			return detail, err
		}
		detail.ChunkProof = proof
		log.Info("prove chunk success", "task-id", task.Task.ID)
		return detail, nil

	case message.ProofTypeBatch:
		proof, err := r.proveBatch(task)
		if err != nil {
			detail.Status = message.StatusProofError
			detail.Error = err.Error()
			return detail, err
		}
		detail.BatchProof = proof
		log.Info("prove batch success", "task-id", task.Task.ID)
		return detail, nil

	default:
		err := fmt.Errorf("invalid task type: %v", task.Task.Type)
		return detail, err
	}
}

func (r *Prover) proveChunk(task *store.ProvingTask) (*message.ChunkProof, error) {
	if task.Task.ChunkTaskDetail == nil {
		return nil, fmt.Errorf("ChunkTaskDetail is empty")
	}
	traces, err := r.getSortedTracesByHashes(task.Task.ChunkTaskDetail.BlockHashes)
	if err != nil {
		return nil, fmt.Errorf("get traces from eth node failed, block hashes: %v, err: %v", task.Task.ChunkTaskDetail.BlockHashes, err)
	}
	chunkTrace := message.ChunkTrace{
		BlockTraces:            traces,
		PrevLastAppliedL1Block: task.Task.ChunkTaskDetail.PrevLastAppliedL1Block,
		L1BlockRangeHash:       task.Task.ChunkTaskDetail.L1BlockRangeHash,
	}
	return r.proverCore.ProveChunk(
		task.Task.ID,
		&chunkTrace,
	)
}

func (r *Prover) proveBatch(task *store.ProvingTask) (*message.BatchProof, error) {
	if task.Task.BatchTaskDetail == nil {
		return nil, fmt.Errorf("BatchTaskDetail is empty")
	}
	return r.proverCore.ProveBatch(task.Task.ID, task.Task.BatchTaskDetail.ChunkInfos, task.Task.BatchTaskDetail.ChunkProofs)
}

func (r *Prover) submitProof(msg *message.ProofDetail, uuid string) error {
	// prepare the submit request
	req := &client.SubmitProofRequest{
		UUID:     uuid,
		TaskID:   msg.ID,
		TaskType: int(msg.Type),
		Status:   int(msg.Status),
	}

	// marshal proof by tasktype
	switch msg.Type {
	case message.ProofTypeChunk:
		if msg.ChunkProof != nil {
			proofData, err := json.Marshal(msg.ChunkProof)
			if err != nil {
				return fmt.Errorf("error marshaling chunk proof: %v", err)
			}
			req.Proof = string(proofData)
		}
	case message.ProofTypeBatch:
		if msg.BatchProof != nil {
			proofData, err := json.Marshal(msg.BatchProof)
			if err != nil {
				return fmt.Errorf("error marshaling batch proof: %v", err)
			}
			req.Proof = string(proofData)
		}
	}

	// send the submit request
	if err := r.coordinatorClient.SubmitProof(r.ctx, req); err != nil {
		if !errors.Is(errors.Unwrap(err), client.ErrCoordinatorConnect) {
			if deleteErr := r.stack.Delete(msg.ID); deleteErr != nil {
				log.Error("prover stack pop failed", "task_type", msg.Type, "task_id", msg.ID, "err", deleteErr)
			}
		}
		return fmt.Errorf("error submitting proof: %v", err)
	}

	if deleteErr := r.stack.Delete(msg.ID); deleteErr != nil {
		log.Error("prover stack pop failed", "task_type", msg.Type, "task_id", msg.ID, "err", deleteErr)
	}
	log.Info("proof submitted successfully", "task-id", msg.ID, "task-type", msg.Type, "task-status", msg.Status, "err", msg.Error)

	return nil
}

func (r *Prover) submitErr(task *store.ProvingTask, proofFailureType message.ProofFailureType, err error) error {
	// prepare the submit request
	req := &client.SubmitProofRequest{
		UUID:        task.Task.UUID,
		TaskID:      task.Task.ID,
		TaskType:    int(task.Task.Type),
		Status:      int(message.StatusProofError),
		Proof:       "",
		FailureType: int(proofFailureType),
		FailureMsg:  err.Error(),
	}

	// send the submit request
	if submitErr := r.coordinatorClient.SubmitProof(r.ctx, req); submitErr != nil {
		if !errors.Is(errors.Unwrap(err), client.ErrCoordinatorConnect) {
			if deleteErr := r.stack.Delete(task.Task.ID); deleteErr != nil {
				log.Error("prover stack pop failed", "task_type", task.Task.Type, "task_id", task.Task.ID, "err", deleteErr)
			}
		}
		return fmt.Errorf("error submitting proof: %v", submitErr)
	}
	if deleteErr := r.stack.Delete(task.Task.ID); deleteErr != nil {
		log.Error("prover stack pop failed", "task_type", task.Task.Type, "task_id", task.Task.ID, "err", deleteErr)
	}

	log.Info("proof submitted report failure successfully",
		"task-id", task.Task.ID, "task-type", task.Task.Type,
		"task-status", message.StatusProofError, "err", err)
	return nil
}

func (r *Prover) getSortedTracesByHashes(blockHashes []common.Hash) ([]*types.BlockTrace, error) {
	if len(blockHashes) == 0 {
		return nil, fmt.Errorf("blockHashes is empty")
	}

	var traces []*types.BlockTrace
	for _, blockHash := range blockHashes {
		trace, err := r.l2GethClient.GetBlockTraceByHash(r.ctx, blockHash)
		if err != nil {
			return nil, err
		}
		traces = append(traces, trace)
	}

	// Sort BlockTraces by header number.
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].Header.Number.Int64() < traces[j].Header.Number.Int64()
	})

	// Check that the block numbers are continuous
	for i := 0; i < len(traces)-1; i++ {
		if traces[i].Header.Number.Int64()+1 != traces[i+1].Header.Number.Int64() {
			return nil, fmt.Errorf("block numbers are not continuous, got %v and %v",
				traces[i].Header.Number.Int64(), traces[i+1].Header.Number.Int64())
		}
	}
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
