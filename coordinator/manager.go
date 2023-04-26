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

	"scroll-tech/database"

	"scroll-tech/common/message"
	"scroll-tech/common/metrics"
	"scroll-tech/common/types"

	"scroll-tech/common/utils/workerpool"

	"scroll-tech/coordinator/config"
	"scroll-tech/coordinator/verifier"
)

var (
	// proofs
	coordinatorProofsReceivedTotalCounter = geth_metrics.NewRegisteredCounter("coordinator/proofs/received/total", metrics.ScrollRegistry)

	coordinatorProofsVerifiedSuccessTimeTimer = geth_metrics.NewRegisteredTimer("coordinator/proofs/verified/success/time", metrics.ScrollRegistry)
	coordinatorProofsVerifiedFailedTimeTimer  = geth_metrics.NewRegisteredTimer("coordinator/proofs/verified/failed/time", metrics.ScrollRegistry)
	coordinatorProofsGeneratedFailedTimeTimer = geth_metrics.NewRegisteredTimer("coordinator/proofs/generated/failed/time", metrics.ScrollRegistry)

	// sessions
	coordinatorSessionsSuccessTotalCounter = geth_metrics.NewRegisteredCounter("coordinator/sessions/success/total", metrics.ScrollRegistry)
	coordinatorSessionsTimeoutTotalCounter = geth_metrics.NewRegisteredCounter("coordinator/sessions/timeout/total", metrics.ScrollRegistry)
	coordinatorSessionsFailedTotalCounter  = geth_metrics.NewRegisteredCounter("coordinator/sessions/failed/total", metrics.ScrollRegistry)

	coordinatorSessionsActiveNumberGauge = geth_metrics.NewRegisteredCounter("coordinator/sessions/active/number", metrics.ScrollRegistry)
)

var (
	ErrNoIdleRoller = errors.New("no roller idle")
)

const (
	proofAndPkBufferSize = 10
)

type rollerProofStatus struct {
	id     string
	typ    message.ProveType
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

	aggTaskChan chan *message.TaskMsg
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
		aggTaskChan:        make(chan *message.TaskMsg, 10),
	}, nil
}

// Start the Manager module.
func (m *Manager) Start() error {
	if m.isRunning() {
		return nil
	}

	m.verifierWorkerPool.Run()

	m.reloadUnassigned()

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
		case taskMsg := <-m.aggTaskChan:
			if taskMsg.SubProofs != nil {
				m.StartProofGenerationSession(taskMsg, nil)
			}
		case <-tick.C:
			// TODO: we should use taskChan instead of db directly
			if len(tasks) == 0 && m.orm != nil {
				var err error
				// TODO: add cache
				if tasks, err = m.orm.GetBlockBatches(
					map[string]interface{}{"proving_status": types.ProvingTaskUnassigned},
					fmt.Sprintf(
						"ORDER BY index %s LIMIT %d;",
						m.cfg.OrderSession,
						m.GetNumberOfIdleRollers(message.BasicProve),
					),
				); err != nil {
					log.Error("failed to get unassigned proving tasks", "error", err)
					continue
				}
			}
			// Select roller and send message

			for len(tasks) > 0 {
				taskID := tasks[0].Hash
				// Get L2 block-traces
				traces, err := m.GetBlockTraces(taskID)
				if err != nil {
					break
				}
				taskMsg := &message.TaskMsg{
					ID:     taskID,
					Type:   message.BasicProve,
					Traces: traces,
				}
				err = m.StartProofGenerationSession(taskMsg, nil)
				if err != nil {
					break
				}
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

func (m *Manager) reloadUnassigned() {
	if m.orm == nil {
		return
	}
	// load unassigned agg tasks into channel
	unassignedAggs, err := m.orm.GetUnassignedAggTasks()
	if err != nil {
		log.Error("failed to load unassigned aggregator tasks from db", "error", err)
		return
	}
	go func() {
		for _, unassigned := range unassignedAggs {
			m.aggTaskChan <- &message.TaskMsg{ID: unassigned.ID, SubProofs: unassigned.SubProofs}
		}
	}()
}

func (m *Manager) restorePrevSessions() {
	// m.orm may be nil in scroll tests
	if m.orm == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// load assigned tasks as sessions
	var (
		taskIDs  []string
		taskMsgs []*message.TaskMsg
	)

	aggTasks, err := m.orm.GetAssignedAggTasks()
	if err != nil {
		log.Error("failed to load assigned aggregator tasks from db", "error", err)
		return
	}
	for _, task := range aggTasks {
		taskMsgs = append(taskMsgs, &message.TaskMsg{
			ID:        task.ID,
			Type:      message.AggregatorProve,
			SubProofs: task.SubProofs,
		})
		taskIDs = append(taskIDs, task.ID)
	}

	// get all assigned basic task hash
	batchHashes, err := m.orm.GetAssignedBatchHashes()
	if err != nil {
		log.Error("failed to get assigned batch taskMsgs from db", "error", err)
		return
	}
	for _, batchHash := range batchHashes {
		traces, err := m.GetBlockTraces(batchHash)
		if err != nil {
			log.Error("failed to get block-traces", "error", err)
			return
		}
		taskMsgs = append(taskMsgs, &message.TaskMsg{
			ID:     batchHash,
			Type:   message.BasicProve,
			Traces: traces,
		})
	}
	taskIDs = append(taskIDs, batchHashes...)

	// get all assigned sessions by block-traces(basic proving tasks) and agg-proofs(agg proving tasks)
	prevSessions, err := m.orm.GetSessionInfosByHashes(taskIDs)
	if err != nil {
		log.Error("failed to recover roller session info from db", "error", err)
		return
	}
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
				"prove type", sess.info.ProveType,
				"public key", roller.PublicKey,
				"proof status", roller.Status)
		}
	}
	for _, msg := range taskMsgs {
		sess := m.sessions[msg.ID]
		go m.CollectProofs(sess, msg)
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
	proofTime := time.Since(time.Unix(sess.info.StartTimestamp, 0))
	proofTimeSec := uint64(proofTime.Seconds())

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
		"proof time", proofTimeSec,
	)

	defer func() {
		// TODO: maybe we should use db tx for the whole process?
		// Roll back current proof's status.
		if dbErr != nil {
			if err := m.updateProofStatus(msg.ID, msg.Type, types.ProvingTaskUnassigned, msg.Proof); err != nil {
				log.Error("fail to reset task status as Unassigned", "msg.ID", msg.ID)
			}
		}
		// set proof status
		status := types.RollerProofInvalid
		if success && dbErr == nil {
			status = types.RollerProofValid
		}
		// notify the session that the roller finishes the proving process
		sess.finishChan <- rollerProofStatus{msg.ID, msg.Type, pk, status}
	}()

	if msg.Status != message.StatusOk {
		coordinatorProofsGeneratedFailedTimeTimer.Update(proofTime)
		m.updateMetricRollerProofsGeneratedFailedTimeTimer(roller.PublicKey, proofTime)
		log.Info(
			"proof generated by roller failed",
			"proof id", msg.ID,
			"roller name", roller.Name,
			"roller pk", roller.PublicKey,
			"proof time", proofTimeSec,
			"error", msg.Error,
		)
		return nil
	}

	// store proof content
	if dbErr = m.orm.UpdateProofByHash(m.ctx, msg.ID, msg.Proof.Proof, msg.Proof.FinalPair, proofTimeSec); dbErr != nil {
		log.Error("failed to store proof into db", "error", dbErr)
		return dbErr
	}
	if dbErr = m.updateProofStatus(msg.ID, msg.Type, types.ProvingTaskProved, msg.Proof); dbErr != nil {
		log.Error("failed to update task status as proved", "error", dbErr)
		return dbErr
	}

	coordinatorProofsReceivedTotalCounter.Inc(1)

	var verifyErr error
	success, verifyErr = m.verifyProof(msg.Proof)
	if verifyErr != nil {
		// TODO: this is only a temp workaround for testnet, we should return err in real cases
		success = false
		log.Error("Failed to verify zk proof", "proof id", msg.ID, "roller name", roller.Name,
			"roller pk", roller.PublicKey, "proof time", proofTimeSec, "error", verifyErr)
		// TODO: Roller needs to be slashed if proof is invalid.
	}

	if success {
		if dbErr = m.updateProofStatus(msg.ID, msg.Type, types.ProvingTaskVerified, msg.Proof); dbErr != nil {
			log.Error(
				"failed to update proving_status",
				"msg.ID", msg.ID,
				"prove type", msg.Type,
				"status", types.ProvingTaskVerified,
				"error", dbErr)
			return dbErr
		}
		coordinatorProofsVerifiedSuccessTimeTimer.Update(proofTime)
		m.updateMetricRollerProofsVerifiedSuccessTimeTimer(roller.PublicKey, proofTime)
		log.Info("proof verified by coordinator success", "proof id", msg.ID, "roller name", roller.Name,
			"roller pk", roller.PublicKey, "proof time", proofTimeSec)
	} else {
		coordinatorProofsVerifiedFailedTimeTimer.Update(proofTime)
		m.updateMetricRollerProofsVerifiedFailedTimeTimer(roller.PublicKey, proofTime)
		log.Info("proof verified by coordinator failed", "proof id", msg.ID, "roller name", roller.Name,
			"roller pk", roller.PublicKey, "proof time", proofTimeSec, "error", verifyErr)
	}
	return nil
}

// CollectProofs collects proofs corresponding to a proof generation session.
func (m *Manager) CollectProofs(sess *session, taskMsg *message.TaskMsg) {
	coordinatorSessionsActiveNumberGauge.Inc(1)
	defer coordinatorSessionsActiveNumberGauge.Dec(1)

	for {
		select {
		//Execute after timeout, set in config.json. Consider all rollers failed.
		case <-time.After(time.Duration(m.cfg.CollectionTime) * time.Minute):
			// Check if session can be replayed
			if sess.info.Attempts < m.cfg.SessionAttempts {
				err := m.StartProofGenerationSession(taskMsg, sess)
				if err == nil {
					m.mu.Lock()
					for pk := range sess.info.Rollers {
						m.freeTaskIDForRoller(pk, sess.info.ID)
					}
					m.mu.Unlock()
					log.Info("Retrying session", "session id:", sess.info.ID)
					return
				}
			}
			// record failed session.
			errMsg := "proof generation session ended without receiving any valid proofs"
			m.addFailedSession(sess, errMsg)
			log.Warn(errMsg, "session id", sess.info.ID)
			// Set status as skipped.
			// Note that this is only a workaround for testnet here.
			// TODO: In real cases we should reset to orm.ProvingTaskUnassigned
			// so as to re-distribute the task in the future
			if err := m.updateProofStatus(sess.info.ID, sess.info.ProveType, types.ProvingTaskFailed, nil); err != nil {
				log.Error("fail to reset task_status as Unassigned", "id", sess.info.ID, "err", err)
			}
			m.mu.Lock()
			for pk := range sess.info.Rollers {
				m.freeTaskIDForRoller(pk, sess.info.ID)
			}
			delete(m.sessions, sess.info.ID)
			m.mu.Unlock()
			coordinatorSessionsTimeoutTotalCounter.Inc(1)
			return

		//Execute after one of the roller finishes sending proof, return early if all rollers had sent results.
		case ret := <-sess.finishChan:
			m.mu.Lock()
			sess.info.Rollers[ret.pk].Status = ret.status
			if sess.isSessionFailed() {
				if err := m.updateProofStatus(ret.id, ret.typ, types.ProvingTaskFailed, nil); err != nil {
					log.Error("failed to update proving_status as failed", "msg.ID", ret.id, "prove type", ret.typ, "error", err)
				}
				coordinatorSessionsFailedTotalCounter.Inc(1)
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

				for pk := range sess.info.Rollers {
					m.freeTaskIDForRoller(pk, sess.info.ID)
				}
				delete(m.sessions, sess.info.ID)
				m.mu.Unlock()

				coordinatorSessionsSuccessTotalCounter.Inc(1)
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

// StartProofGenerationSession starts a common proof generation session
func (m *Manager) StartProofGenerationSession(taskMsg *message.TaskMsg, prevSession *session) (err error) {
	var (
		taskId        = taskMsg.ID
		taskProveType = taskMsg.Type
	)

	if m.GetNumberOfIdleRollers(taskProveType) == 0 {
		log.Warn("no idle roller when starting proof generation session", "id", taskId, "type", taskProveType)
		return ErrNoIdleRoller
	}

	log.Info("start proof generation session", "id", taskId, "type", taskProveType)
	defer func() {
		if err != nil {
			if taskMsg != nil {
				if err := m.updateProofStatus(taskId, taskProveType, types.ProvingTaskUnassigned, nil); err != nil {
					log.Error("fail to reset task_status as Unassigned", "id", taskId, "err", err)
				}
			} else {
				if err := m.updateProofStatus(taskId, taskProveType, types.ProvingTaskFailed, nil); err != nil {
					log.Error("fail to reset task_status as Failed", "id", taskId, "err", err)
				}
			}
		}
	}()

	// Dispatch taskMsg to rollers.
	rollers, err := m.DispatchTaskToRoller(taskMsg)
	if err != nil {
		return err
	}

	// Update session proving status as assigned.
	if err = m.updateProofStatus(taskId, taskProveType, types.ProvingTaskAssigned, nil); err != nil {
		return err
	}

	// Create a proof generation session.
	sess := &session{
		info: &types.SessionInfo{
			ID:             taskId,
			ProveType:      taskProveType,
			Rollers:        rollers,
			StartTimestamp: time.Now().Unix(),
			Attempts:       1,
		},
		finishChan: make(chan rollerProofStatus, proofAndPkBufferSize),
	}
	if prevSession != nil {
		sess.info.Attempts += prevSession.info.Attempts
	}

	for _, roller := range sess.info.Rollers {
		log.Info(
			"assigned proof to roller",
			"session id", sess.info.ID,
			"prove type", sess.info.ProveType,
			"roller name", roller.Name,
			"roller pk", roller.PublicKey,
			"proof status", roller.Status)
	}

	// Store session info.
	if err = m.orm.SetSessionInfo(sess.info); err != nil {
		log.Error("db set session info fail", "session id", sess.info.ID, "error", err)
		return err
	}

	m.mu.Lock()
	m.sessions[taskId] = sess
	m.mu.Unlock()
	go m.CollectProofs(sess, taskMsg)

	return nil
}

// GetBlockTraces get L2 block-traces by batch-hash from ethereum.
func (m *Manager) GetBlockTraces(batchHash string) ([]*geth_types.BlockTrace, error) {
	blockInfos, err := m.orm.GetL2BlockInfos(map[string]interface{}{"batch_hash": batchHash})
	if err != nil {
		log.Error(
			"could not GetBlockInfos",
			"batch_hash", batchHash,
			"error", err,
		)
		return nil, err
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
			return nil, err
		}
	}
	return traces, nil
}

// DispatchTaskToRoller dispatches task to basic/aggregator rollers.
func (m *Manager) DispatchTaskToRoller(taskMsg *message.TaskMsg) (map[string]*types.RollerStatus, error) {
	rollers := make(map[string]*types.RollerStatus)
	for i := 0; i < int(m.cfg.RollersPerSession); i++ {
		roller := m.selectRoller(taskMsg.Type)
		if roller == nil {
			log.Info("selectRoller returns nil")
			break
		}
		log.Info("roller is picked", "session id", taskMsg.ID, "name", roller.Name, "prove type", taskMsg.Type, "public key", roller.PublicKey)
		// send trace to roller
		if !roller.sendTask(taskMsg) {
			log.Error("send task failed", "roller name", roller.Name, "prove type", taskMsg.Type, "public key", roller.PublicKey, "id", taskMsg.ID)
			continue
		}
		m.updateMetricRollerProofsLastAssignedTimestampGauge(roller.PublicKey)
		rollers[roller.PublicKey] = &types.RollerStatus{PublicKey: roller.PublicKey, Name: roller.Name, Status: types.RollerAssigned}
	}
	// No roller assigned.
	if len(rollers) == 0 {
		log.Error("no roller assigned", "id", taskMsg.ID, "type", taskMsg.Type, "number of idle rollers", m.GetNumberOfIdleRollers(taskMsg.Type))
		return nil, ErrNoIdleRoller
	}
	return rollers, nil
}

func (m *Manager) addFailedSession(sess *session, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedSessionInfos[sess.info.ID] = newSessionInfo(sess, types.ProvingTaskFailed, errMsg, true)
}

func (m *Manager) updateProofStatus(taskId string, proveType message.ProveType, status types.ProvingStatus, proof *message.AggProof) (err error) {
	switch proveType {
	case message.BasicProve:
		if err = m.orm.UpdateProvingStatus(taskId, status); err != nil {
			log.Error("failed to update basic taskMsg status", "id", taskId, "err", err)
		}
	case message.AggregatorProve:
		if err = m.orm.UpdateAggTaskStatus(taskId, status, proof); err != nil {
			log.Error("failed to update aggregator taskMsg status", "id", taskId, "err", err)
		}
	}
	return
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
