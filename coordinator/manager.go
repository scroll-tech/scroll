package coordinator

import (
	"context"
	"errors"
	"fmt"
	mathrand "math/rand"
	"sync"
	"sync/atomic"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/patrickmn/go-cache"
	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	geth_metrics "github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/message"
	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	"scroll-tech/database"

	"scroll-tech/common/utils/workerpool"

	"scroll-tech/coordinator/config"
	"scroll-tech/coordinator/verifier"
)

var (
	coordinatorSessionsTimeoutTotalCounter      = geth_metrics.NewRegisteredCounter("coordinator/sessions/timeout/total", metrics.ScrollRegistry)
	coordinatorProofsReceivedTotalCounter       = geth_metrics.NewRegisteredCounter("coordinator/proofs/received/total", metrics.ScrollRegistry)
	coordinatorProofsVerifiedTotalCounter       = geth_metrics.NewRegisteredCounter("coordinator/proofs/verified/total", metrics.ScrollRegistry)
	coordinatorProofsVerifiedFailedTotalCounter = geth_metrics.NewRegisteredCounter("coordinator/proofs/verified/failed/total", metrics.ScrollRegistry)
)

const (
	proofAndPkBufferSize = 10
)

type rollerProofStatus struct {
	id     string
	pk     string
	status types.RollerProveStatus
}

// Contains all the information on an ongoing proof generation session.
type session struct {
	info *types.SessionInfo
	// finish channel is used to pass the public key of the rollers who finished proving process.
	finishChan chan rollerProofStatus
}

// Manager is responsible for maintaining connections with active rollers,
// sending the challenges, and receiving proofs. It also regulates the reward
// distribution. All read and write logic and connection handling happens through
// a modular websocket server, contained within the Manager. Incoming messages are
// then passed to the Manager where the actual handling logic resides.
type Manager struct {
	// The manager context.
	ctx context.Context

	// The roller manager configuration.
	cfg *config.RollerManagerConfig

	// The indicator whether the backend is running or not.
	running int32

	// A mutex guarding the boolean below.
	mu sync.RWMutex
	// A map containing all active proof generation sessions.
	sessions map[string]*session
	// A map containing proof failed or verify failed proof.
	rollerPool cmap.ConcurrentMap

	failedSessionInfos map[string]*SessionInfo

	// A direct connection to the Halo2 verifier, used to verify
	// incoming proofs.
	verifier *verifier.Verifier

	// db interface
	orm database.OrmFactory

	// l2geth client
	*ethclient.Client

	// Token cache
	tokenCache *cache.Cache
	// A mutex guarding registration
	registerMu sync.RWMutex

	// Verifier worker pool
	verifierWorkerPool *workerpool.WorkerPool
}

// New returns a new instance of Manager. The instance will be not fully prepared,
// and still needs to be finalized and ran by calling `manager.Start`.
func New(ctx context.Context, cfg *config.RollerManagerConfig, orm database.OrmFactory, client *ethclient.Client) (*Manager, error) {
	v, err := verifier.NewVerifier(cfg.Verifier)
	if err != nil {
		return nil, err
	}

	log.Info("Start coordinator successfully.")
	return &Manager{
		ctx:                ctx,
		cfg:                cfg,
		rollerPool:         cmap.New(),
		sessions:           make(map[string]*session),
		failedSessionInfos: make(map[string]*SessionInfo),
		verifier:           v,
		orm:                orm,
		Client:             client,
		tokenCache:         cache.New(time.Duration(cfg.TokenTimeToLive)*time.Second, 1*time.Hour),
		verifierWorkerPool: workerpool.NewWorkerPool(cfg.MaxVerifierWorkers),
	}, nil
}

// Start the Manager module.
func (m *Manager) Start() error {
	if m.isRunning() {
		return nil
	}

	m.verifierWorkerPool.Run()
	m.restorePrevSessions()

	atomic.StoreInt32(&m.running, 1)

	go m.Loop()
	return nil
}

// Stop the Manager module, for a graceful shutdown.
func (m *Manager) Stop() {
	if !m.isRunning() {
		return
	}
	m.verifierWorkerPool.Stop()

	atomic.StoreInt32(&m.running, 0)
}

// isRunning returns an indicator whether manager is running or not.
func (m *Manager) isRunning() bool {
	return atomic.LoadInt32(&m.running) == 1
}

// Loop keeps the manager running.
func (m *Manager) Loop() {
	var (
		tick  = time.NewTicker(time.Second * 2)
		tasks []*types.BlockBatch
	)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if len(tasks) == 0 && m.orm != nil {
				var err error
				// TODO: add cache
				if tasks, err = m.orm.GetBlockBatches(
					map[string]interface{}{"proving_status": types.ProvingTaskUnassigned},
					fmt.Sprintf(
						"ORDER BY index %s LIMIT %d;",
						m.cfg.OrderSession,
						m.GetNumberOfIdleRollers(),
					),
				); err != nil {
					log.Error("failed to get unassigned proving tasks", "error", err)
					continue
				}
			}
			// Select roller and send message
			for len(tasks) > 0 && m.StartProofGenerationSession(tasks[0]) {
				tasks = tasks[1:]
			}
		case <-m.ctx.Done():
			if m.ctx.Err() != nil {
				log.Error(
					"manager context canceled with error",
					"error", m.ctx.Err(),
				)
			}
			return
		}
	}
}

func (m *Manager) restorePrevSessions() {
	// m.orm may be nil in scroll tests
	if m.orm == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if hashes, err := m.orm.GetAssignedBatchHashes(); err != nil {
		log.Error("failed to get assigned batch hashes from db", "error", err)
	} else if prevSessions, err := m.orm.GetSessionInfosByHashes(hashes); err != nil {
		log.Error("failed to recover roller session info from db", "error", err)
	} else {
		for _, v := range prevSessions {
			sess := &session{
				info:       v,
				finishChan: make(chan rollerProofStatus, proofAndPkBufferSize),
			}
			m.sessions[sess.info.ID] = sess

			log.Info("Coordinator restart reload sessions", "session start time", time.Unix(sess.info.StartTimestamp, 0))
			for _, roller := range sess.info.Rollers {
				log.Info(
					"restore roller info for session",
					"session id", sess.info.ID,
					"roller name", roller.Name,
					"public key", roller.PublicKey,
					"proof status", roller.Status)
			}

			go m.CollectProofs(sess)

			id := sess.info.ID
			batches, err := m.orm.GetBlockBatches(map[string]interface{}{"hash": id})
			if err != nil || len(batches) == 0 {
				log.Error("Failed to GetBlockBatches", "batch_hash", id, "err", err)
				continue
			}
			for i := range batches {
				go m.tryVerify(sess, batches[i])
			}
		}
	}
}

// TryVerify verifies a proof whose verification was previously interrupted by a crash or restart
func (m *Manager) tryVerify(sess *session, batch *types.BlockBatch) {
	if batch.ProvingStatus != types.ProvingTaskProved {
		return
	}
	var success bool
	var dbErr error

	proof := &message.AggProof{
		Proof:     batch.Proof,
		Instance:  batch.InstanceCommitments,
		FinalPair: batch.FinalPair,
		Vk:        batch.Vk,
	}
	defer func() {
		if dbErr != nil {
			if err := m.orm.UpdateProvingStatus(batch.Hash, types.ProvingTaskUnassigned); err != nil {
				log.Error("fail to reset task status as Unassigned", "msg.ID", batch.Hash)
			}
		}
		// set proof status
		status := types.RollerProofInvalid
		if success && dbErr == nil {
			status = types.RollerProofValid
		}

		// TODO: chnge this
		// Currently we have only one roller per session, so the following will work
		var pk string
		for _, roller := range sess.info.Rollers {
			pk = roller.PublicKey
		}

		// notify the session that the roller finishes the proving process
		sess.finishChan <- rollerProofStatus{batch.Hash, pk, status}
	}()

	success, err := m.verifier.VerifyProof(proof)
	if err != nil {
		success = false
		log.Error("Failed to verify zk proof", "proof id", batch.Hash, "error", err)
		return
	} else {
		log.Info("Verify zk proof successfully", "verification result", success, "proof id", batch.Hash)
	}
	if dbErr = m.orm.UpdateProvingStatus(batch.Hash, types.ProvingTaskVerified); dbErr != nil {
		log.Error(
			"failed to update proving_status",
			"msg.ID", batch.Hash,
			"status", types.ProvingTaskVerified,
			"error", dbErr)
	}
}

// HandleZkProof handle a ZkProof submitted from a roller.
// For now only proving/verifying error will lead to setting status as skipped.
// db/unmarshal errors will not because they are errors on the business logic side.
func (m *Manager) handleZkProof(pk string, msg *message.ProofDetail) error {
	var dbErr error
	var success bool

	// Assess if the proof generation session for the given ID is still active.
	// We hold the read lock until the end of the function so that there is no
	// potential race for channel deletion.
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[msg.ID]
	if !ok {
		return fmt.Errorf("proof generation session for id %v does not existID", msg.ID)
	}
	proofTimeSec := uint64(time.Since(time.Unix(sess.info.StartTimestamp, 0)).Seconds())

	// Ensure this roller is eligible to participate in the session.
	roller, ok := sess.info.Rollers[pk]
	if !ok {
		return fmt.Errorf("roller %s (%s) is not eligible to partake in proof session %v", roller.Name, roller.PublicKey, msg.ID)
	}
	if roller.Status == types.RollerProofValid {
		// In order to prevent DoS attacks, it is forbidden to repeatedly submit valid proofs.
		// TODO: Defend invalid proof resubmissions by one of the following two methods:
		// (i) slash the roller for each submission of invalid proof
		// (ii) set the maximum failure retry times
		log.Warn(
			"roller has already submitted valid proof in proof session",
			"roller name", roller.Name,
			"roller pk", roller.PublicKey,
			"proof id", msg.ID,
		)
		return nil
	}
	log.Info(
		"handling zk proof",
		"proof id", msg.ID,
		"roller name", roller.Name,
		"roller pk", roller.PublicKey,
	)

	defer func() {
		// TODO: maybe we should use db tx for the whole process?
		// Roll back current proof's status.
		if dbErr != nil {
			if err := m.orm.UpdateProvingStatus(msg.ID, types.ProvingTaskUnassigned); err != nil {
				log.Error("fail to reset task status as Unassigned", "msg.ID", msg.ID)
			}
		}
		// set proof status
		status := types.RollerProofInvalid
		if success && dbErr == nil {
			status = types.RollerProofValid
		}
		// notify the session that the roller finishes the proving process
		sess.finishChan <- rollerProofStatus{msg.ID, pk, status}
	}()

	if msg.Status != message.StatusOk {
		log.Error(
			"Roller failed to generate proof",
			"msg.ID", msg.ID,
			"roller name", roller.Name,
			"roller pk", roller.PublicKey,
			"error", msg.Error,
		)
		return nil
	}

	// store proof content
	if dbErr = m.orm.UpdateProofByHash(m.ctx, msg.ID, msg.Proof.Proof, msg.Proof.Instance, msg.Proof.FinalPair, msg.Proof.Vk, proofTimeSec); dbErr != nil {
		log.Error("failed to store proof into db", "error", dbErr)
		return dbErr
	}
	if dbErr = m.orm.UpdateProvingStatus(msg.ID, types.ProvingTaskProved); dbErr != nil {
		log.Error("failed to update task status as proved", "error", dbErr)
		return dbErr
	}

	coordinatorProofsReceivedTotalCounter.Inc(1)

	var err error
	success, err = m.verifyProof(msg.Proof)
	if err != nil {
		// TODO: this is only a temp workaround for testnet, we should return err in real cases
		success = false
		log.Error("Failed to verify zk proof", "proof id", msg.ID, "error", err)
		// TODO: Roller needs to be slashed if proof is invalid.
	} else {
		log.Info("Verify zk proof successfully", "verification result", success, "proof id", msg.ID)
	}

	if success {
		if dbErr = m.orm.UpdateProvingStatus(msg.ID, types.ProvingTaskVerified); dbErr != nil {
			log.Error(
				"failed to update proving_status",
				"msg.ID", msg.ID,
				"status", types.ProvingTaskVerified,
				"error", dbErr)
			return dbErr
		}
		coordinatorProofsVerifiedTotalCounter.Inc(1)
	} else {
		coordinatorProofsVerifiedFailedTotalCounter.Inc(1)
	}
	return nil
}

// CollectProofs collects proofs corresponding to a proof generation session.
func (m *Manager) CollectProofs(sess *session) {
	//Cleanup roller sessions before return.
	defer func() {
		// TODO: remove the clean-up, rollers report healthy status.
		m.mu.Lock()
		for pk := range sess.info.Rollers {
			m.freeTaskIDForRoller(pk, sess.info.ID)
		}
		delete(m.sessions, sess.info.ID)
		m.mu.Unlock()
	}()
	for {
		select {
		//Execute after timeout, set in config.json. Consider all rollers failed.
		case <-time.After(time.Duration(m.cfg.CollectionTime) * time.Minute):
			// record failed session.
			errMsg := "proof generation session ended without receiving any valid proofs"
			m.addFailedSession(sess, errMsg)
			log.Warn(errMsg, "session id", sess.info.ID)
			// Set status as skipped.
			// Note that this is only a workaround for testnet here.
			// TODO: In real cases we should reset to orm.ProvingTaskUnassigned
			// so as to re-distribute the task in the future
			if err := m.orm.UpdateProvingStatus(sess.info.ID, types.ProvingTaskFailed); err != nil {
				log.Error("fail to reset task_status as Unassigned", "id", sess.info.ID, "err", err)
			}
			return

		//Execute after one of the roller finishes sending proof, return early if all rollers had sent results.
		case ret := <-sess.finishChan:
			m.mu.Lock()
			sess.info.Rollers[ret.pk].Status = ret.status
			if sess.isSessionFailed() {
				if err := m.orm.UpdateProvingStatus(ret.id, types.ProvingTaskFailed); err != nil {
					log.Error("failed to update proving_status as failed", "msg.ID", ret.id, "error", err)
				}
			}
			if err := m.orm.SetSessionInfo(sess.info); err != nil {
				log.Error("db set session info fail", "pk", ret.pk, "error", err)
			}
			//Check if all rollers have finished their tasks, and rollers with valid results are indexed by public key.
			finished, validRollers := sess.isRollersFinished()

			//When all rollers have finished submitting their tasks, select a winner within rollers with valid proof, and return, terminate the for loop.
			if finished && len(validRollers) > 0 {
				//Select a random index for this slice.
				randIndex := mathrand.Intn(len(validRollers))
				_ = validRollers[randIndex]
				// TODO: reward winner
				m.mu.Unlock()
				return
			}
			m.mu.Unlock()
		}
	}
}

// isRollersFinished checks if all rollers have finished submitting proofs, check their validity, and record rollers who produce valid proof.
// When rollersLeft reaches 0, it means all rollers have finished their tasks.
// validRollers also records the public keys of rollers who have finished their tasks correctly as index.
func (s *session) isRollersFinished() (bool, []string) {
	var validRollers []string
	for pk, roller := range s.info.Rollers {
		if roller.Status == types.RollerProofValid {
			validRollers = append(validRollers, pk)
			continue
		}
		if roller.Status == types.RollerProofInvalid {
			continue
		}
		// Some rollers are still proving.
		return false, nil
	}
	return true, validRollers
}

func (s *session) isSessionFailed() bool {
	for _, roller := range s.info.Rollers {
		if roller.Status != types.RollerProofInvalid {
			return false
		}
	}
	return true
}

// APIs collect API services.
func (m *Manager) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "roller",
			Service:   RollerAPI(m),
			Public:    true,
		},
		{
			Namespace: "debug",
			Public:    true,
			Service:   RollerDebugAPI(m),
		},
	}
}

// StartProofGenerationSession starts a proof generation session
func (m *Manager) StartProofGenerationSession(task *types.BlockBatch) (success bool) {
	if m.GetNumberOfIdleRollers() == 0 {
		log.Warn("no idle roller when starting proof generation session", "id", task.Hash)
		return false
	}

	log.Info("start proof generation session", "id", task.Hash)
	defer func() {
		if !success {
			if err := m.orm.UpdateProvingStatus(task.Hash, types.ProvingTaskUnassigned); err != nil {
				log.Error("fail to reset task_status as Unassigned", "id", task.Hash, "err", err)
			}
		}
	}()

	// Get block traces.
	blockInfos, err := m.orm.GetL2BlockInfos(map[string]interface{}{"batch_hash": task.Hash})
	if err != nil {
		log.Error(
			"could not GetBlockInfos",
			"batch_hash", task.Hash,
			"error", err,
		)
		return false
	}
	traces := make([]*geth_types.BlockTrace, len(blockInfos))
	for i, blockInfo := range blockInfos {
		traces[i], err = m.Client.GetBlockTraceByHash(m.ctx, common.HexToHash(blockInfo.Hash))
		if err != nil {
			log.Error(
				"could not GetBlockTraceByNumber",
				"block number", blockInfo.Number,
				"block hash", blockInfo.Hash,
				"error", err,
			)
			return false
		}
	}

	// Dispatch task to rollers.
	rollers := make(map[string]*types.RollerStatus)
	for i := 0; i < int(m.cfg.RollersPerSession); i++ {
		roller := m.selectRoller()
		if roller == nil {
			log.Info("selectRoller returns nil")
			break
		}
		log.Info("roller is picked", "session id", task.Hash, "name", roller.Name, "public key", roller.PublicKey)
		// send trace to roller
		if !roller.sendTask(task.Hash, traces) {
			log.Error("send task failed", "roller name", roller.Name, "public key", roller.PublicKey, "id", task.Hash)
			continue
		}
		rollers[roller.PublicKey] = &types.RollerStatus{PublicKey: roller.PublicKey, Name: roller.Name, Status: types.RollerAssigned}
	}
	// No roller assigned.
	if len(rollers) == 0 {
		log.Error("no roller assigned", "id", task.Hash, "number of idle rollers", m.GetNumberOfIdleRollers())
		return false
	}

	// Update session proving status as assigned.
	if err = m.orm.UpdateProvingStatus(task.Hash, types.ProvingTaskAssigned); err != nil {
		log.Error("failed to update task status", "id", task.Hash, "err", err)
		return false
	}

	// Create a proof generation session.
	sess := &session{
		info: &types.SessionInfo{
			ID:             task.Hash,
			Rollers:        rollers,
			StartTimestamp: time.Now().Unix(),
		},
		finishChan: make(chan rollerProofStatus, proofAndPkBufferSize),
	}

	// Store session info.
	if err = m.orm.SetSessionInfo(sess.info); err != nil {
		log.Error("db set session info fail", "error", err)
		for _, roller := range sess.info.Rollers {
			log.Error(
				"restore roller info for session",
				"session id", sess.info.ID,
				"roller name", roller.Name,
				"public key", roller.PublicKey,
				"proof status", roller.Status)
		}
		return false
	}

	m.mu.Lock()
	m.sessions[task.Hash] = sess
	m.mu.Unlock()
	go m.CollectProofs(sess)

	return true
}

// IsRollerIdle determines whether this roller is idle.
func (m *Manager) IsRollerIdle(hexPk string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// We need to iterate over all sessions because finished sessions will be deleted until the
	// timeout. So a busy roller could be marked as idle in a finished session.
	for _, sess := range m.sessions {
		for pk, roller := range sess.info.Rollers {
			if pk == hexPk && roller.Status == types.RollerAssigned {
				return false
			}
		}
	}

	return true
}

func (m *Manager) addFailedSession(sess *session, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedSessionInfos[sess.info.ID] = newSessionInfo(sess, types.ProvingTaskFailed, errMsg, true)
}

// VerifyToken verifies pukey for token and expiration time
func (m *Manager) VerifyToken(authMsg *message.AuthMsg) (bool, error) {
	pubkey, _ := authMsg.PublicKey()
	// GetValue returns nil if value is expired
	if token, ok := m.tokenCache.Get(pubkey); !ok || token != authMsg.Identity.Token {
		return false, errors.New("failed to find corresponding token")
	}
	return true, nil
}

func (m *Manager) addVerifyTask(proof *message.AggProof) chan verifyResult {
	c := make(chan verifyResult, 1)
	m.verifierWorkerPool.AddTask(func() {
		result, err := m.verifier.VerifyProof(proof)
		c <- verifyResult{result, err}
	})
	return c
}

func (m *Manager) verifyProof(proof *message.AggProof) (bool, error) {
	if !m.isRunning() {
		return false, errors.New("coordinator has stopped before verification")
	}
	verifyResultChan := m.addVerifyTask(proof)
	verifyResult := <-verifyResultChan
	return verifyResult.result, verifyResult.err
}

type verifyResult struct {
	result bool
	err    error
}
