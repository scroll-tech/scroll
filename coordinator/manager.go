package coordinator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/patrickmn/go-cache"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	geth_metrics "github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rpc"
	"golang.org/x/exp/rand"
	"gorm.io/gorm"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils/workerpool"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
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

const (
	proofAndPkBufferSize = 10
)

type rollerProofStatus struct {
	id     string
	typ    message.ProofType
	pk     string
	status types.RollerProveStatus
}

// Contains all the information on an ongoing proof generation session.
type session struct {
	taskID       string
	sessionInfos []*orm.SessionInfo
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

	// orm interface
	l2BlockOrm     *orm.L2Block
	chunkOrm       *orm.Chunk
	batchOrm       *orm.Batch
	sessionInfoOrm *orm.SessionInfo

	// Token cache
	tokenCache *cache.Cache
	// A mutex guarding registration
	registerMu sync.RWMutex

	// Verifier worker pool
	verifierWorkerPool *workerpool.WorkerPool
}

// New returns a new instance of Manager. The instance will be not fully prepared,
// and still needs to be finalized and ran by calling `manager.Start`.
func New(ctx context.Context, cfg *config.RollerManagerConfig, db *gorm.DB) (*Manager, error) {
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
		l2BlockOrm:         orm.NewL2Block(db),
		chunkOrm:           orm.NewChunk(db),
		batchOrm:           orm.NewBatch(db),
		sessionInfoOrm:     orm.NewSessionInfo(db),
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
		tick       = time.NewTicker(time.Second * 2)
		chunkTasks []*orm.Chunk
		batchTasks []*orm.Batch
	)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			// load and send batch tasks
			if len(batchTasks) == 0 {
				var err error
				batchTasks, err = m.batchOrm.GetUnassignedBatches(m.ctx, m.GetNumberOfIdleRollers(message.ProofTypeBatch))
				if err != nil {
					log.Error("failed to get unassigned batch proving tasks", "error", err)
					continue
				}
			}
			// Select batch type roller and send message
			for len(batchTasks) > 0 && m.StartBatchProofGenerationSession(batchTasks[0], nil) {
				batchTasks = batchTasks[1:]
			}

			// load and send chunk tasks
			if len(chunkTasks) == 0 {
				// TODO: add cache
				var err error
				chunkTasks, err = m.chunkOrm.GetUnassignedChunks(m.ctx, m.GetNumberOfIdleRollers(message.ProofTypeChunk))
				if err != nil {
					log.Error("failed to get unassigned chunk proving tasks", "error", err)
					continue
				}
			}
			// Select chunk type roller and send message
			for len(chunkTasks) > 0 && m.StartChunkProofGenerationSession(chunkTasks[0], nil) {
				chunkTasks = chunkTasks[1:]
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
	m.mu.Lock()
	defer m.mu.Unlock()

	var hashes []string
	// load assigned batch tasks from db
	batchTasks, err := m.batchOrm.GetAssignedBatches(m.ctx)
	if err != nil {
		log.Error("failed to load assigned batch tasks from db", "error", err)
		return
	}
	for _, batchTask := range batchTasks {
		hashes = append(hashes, batchTask.Hash)
	}
	// load assigned chunk tasks from db
	chunkTasks, err := m.chunkOrm.GetAssignedChunks(m.ctx)
	if err != nil {
		log.Error("failed to get assigned batch batchHashes from db", "error", err)
		return
	}
	for _, chunkTask := range chunkTasks {
		hashes = append(hashes, chunkTask.Hash)
	}
	prevSessions, err := m.sessionInfoOrm.GetSessionInfosByHashes(m.ctx, hashes)
	if err != nil {
		log.Error("failed to recover roller session info from db", "error", err)
		return
	}

	sessionInfosMaps := make(map[string][]*orm.SessionInfo)
	for _, v := range prevSessions {
		log.Info("restore roller info for session", "session start time", v.CreatedAt, "session id", v.TaskID, "roller name",
			v.RollerName, "proof type", v.ProofType, "public key", v.RollerPublicKey, "proof status", v.ProvingStatus)
		sessionInfosMaps[v.TaskID] = append(sessionInfosMaps[v.TaskID], v)
	}

	for taskID, sessionInfos := range sessionInfosMaps {
		sess := &session{
			taskID:       taskID,
			sessionInfos: sessionInfos,
			finishChan:   make(chan rollerProofStatus, proofAndPkBufferSize),
		}
		m.sessions[taskID] = sess
		go m.CollectProofs(sess)
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

	var tmpSessionInfo *orm.SessionInfo
	for _, si := range sess.sessionInfos {
		// get the send session info of this proof msg
		if si.TaskID == msg.ID && si.RollerPublicKey == pk {
			tmpSessionInfo = si
		}
	}

	if tmpSessionInfo == nil {
		return fmt.Errorf("proof generation session for id %v pk:%s does not existID", msg.ID, pk)
	}

	proofTime := time.Since(tmpSessionInfo.CreatedAt)
	proofTimeSec := uint64(proofTime.Seconds())

	// Ensure this roller is eligible to participate in the session.
	if types.RollerProveStatus(tmpSessionInfo.ProvingStatus) == types.RollerProofValid {
		// In order to prevent DoS attacks, it is forbidden to repeatedly submit valid proofs.
		// TODO: Defend invalid proof resubmissions by one of the following two methods:
		// (i) slash the roller for each submission of invalid proof
		// (ii) set the maximum failure retry times
		log.Warn(
			"roller has already submitted valid proof in proof session",
			"roller name", tmpSessionInfo.RollerName,
			"roller pk", tmpSessionInfo.RollerPublicKey,
			"proof type", tmpSessionInfo.ProofType,
			"proof id", msg.ID,
		)
		return nil
	}

	log.Info("handling zk proof", "proof id", msg.ID, "roller name", tmpSessionInfo.RollerName, "roller pk",
		tmpSessionInfo.RollerPublicKey, "proof type", tmpSessionInfo.ProofType, "proof time", proofTimeSec)

	defer func() {
		// TODO: maybe we should use db tx for the whole process?
		// Roll back current proof's status.
		if dbErr != nil {
			if msg.Type == message.ProofTypeChunk {
				if err := m.chunkOrm.UpdateProvingStatus(m.ctx, msg.ID, types.ProvingTaskUnassigned); err != nil {
					log.Error("fail to reset chunk task status as Unassigned", "msg.ID", msg.ID)
				}
			}
			if msg.Type == message.ProofTypeBatch {
				if err := m.batchOrm.UpdateProvingStatus(m.ctx, msg.ID, types.ProvingTaskUnassigned); err != nil {
					log.Error("fail to reset batch task status as Unassigned", "msg.ID", msg.ID)
				}
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
		m.updateMetricRollerProofsGeneratedFailedTimeTimer(tmpSessionInfo.RollerPublicKey, proofTime)
		log.Info(
			"proof generated by roller failed",
			"proof id", msg.ID,
			"roller name", tmpSessionInfo.RollerName,
			"roller pk", tmpSessionInfo.RollerPublicKey,
			"proof type", msg.Type,
			"proof time", proofTimeSec,
			"error", msg.Error,
		)
		return nil
	}

	// store proof content
	if msg.Type == message.ProofTypeChunk {
		if dbErr = m.chunkOrm.UpdateProofByHash(m.ctx, msg.ID, msg.Proof, proofTimeSec); dbErr != nil {
			log.Error("failed to store chunk proof into db", "error", dbErr)
			return dbErr
		}
		if dbErr = m.chunkOrm.UpdateProvingStatus(m.ctx, msg.ID, types.ProvingTaskProved); dbErr != nil {
			log.Error("failed to update chunk task status as proved", "error", dbErr)
			return dbErr
		}
	}
	if msg.Type == message.ProofTypeBatch {
		if dbErr = m.batchOrm.UpdateProofByHash(m.ctx, msg.ID, msg.Proof, proofTimeSec); dbErr != nil {
			log.Error("failed to store batch proof into db", "error", dbErr)
			return dbErr
		}
		if dbErr = m.batchOrm.UpdateProvingStatus(m.ctx, msg.ID, types.ProvingTaskProved); dbErr != nil {
			log.Error("failed to update batch task status as proved", "error", dbErr)
			return dbErr
		}
	}

	coordinatorProofsReceivedTotalCounter.Inc(1)

	var verifyErr error
	// TODO: wrap both chunk verifier and batch verifier
	success, verifyErr = m.verifyProof(msg.Proof)
	if verifyErr != nil {
		// TODO: this is only a temp workaround for testnet, we should return err in real cases
		success = false
		log.Error("Failed to verify zk proof", "proof id", msg.ID, "roller name", tmpSessionInfo.RollerName,
			"roller pk", tmpSessionInfo.RollerPublicKey, "proof type", msg.Type, "proof time", proofTimeSec, "error", verifyErr)
		// TODO: Roller needs to be slashed if proof is invalid.
	}

	if success {
		if msg.Type == message.ProofTypeChunk {
			if dbErr = m.chunkOrm.UpdateProvingStatus(m.ctx, msg.ID, types.ProvingTaskVerified); dbErr != nil {
				log.Error(
					"failed to update chunk proving_status",
					"msg.ID", msg.ID,
					"status", types.ProvingTaskVerified,
					"error", dbErr)
				return dbErr
			}
		}
		if msg.Type == message.ProofTypeBatch {
			if dbErr = m.batchOrm.UpdateProvingStatus(m.ctx, msg.ID, types.ProvingTaskVerified); dbErr != nil {
				log.Error(
					"failed to update batch proving_status",
					"msg.ID", msg.ID,
					"status", types.ProvingTaskVerified,
					"error", dbErr)
				return dbErr
			}
		}

		coordinatorProofsVerifiedSuccessTimeTimer.Update(proofTime)
		m.updateMetricRollerProofsVerifiedSuccessTimeTimer(tmpSessionInfo.RollerPublicKey, proofTime)
		log.Info("proof verified by coordinator success", "proof id", msg.ID, "roller name", tmpSessionInfo.RollerName,
			"roller pk", tmpSessionInfo.RollerPublicKey, "proof type", msg.Type, "proof time", proofTimeSec)
	} else {
		coordinatorProofsVerifiedFailedTimeTimer.Update(proofTime)
		m.updateMetricRollerProofsVerifiedFailedTimeTimer(tmpSessionInfo.RollerPublicKey, proofTime)
		log.Info("proof verified by coordinator failed", "proof id", msg.ID, "roller name", tmpSessionInfo.RollerName,
			"roller pk", tmpSessionInfo.RollerPublicKey, "proof type", msg.Type, "proof time", proofTimeSec, "error", verifyErr)
	}
	return nil
}

// checkAttempts use the count of session info to check the attempts
func (m *Manager) checkAttemptsExceeded(hash string) bool {
	sessionInfos, err := m.sessionInfoOrm.GetSessionInfosByHashes(context.Background(), []string{hash})
	if err != nil {
		log.Error("get session info error", "hash id", hash, "error", err)
		return true
	}

	if len(sessionInfos) >= int(m.cfg.SessionAttempts) {
		return true
	}
	return false
}

// CollectProofs collects proofs corresponding to a proof generation session.
func (m *Manager) CollectProofs(sess *session) {
	coordinatorSessionsActiveNumberGauge.Inc(1)
	defer coordinatorSessionsActiveNumberGauge.Dec(1)

	for {
		select {
		//Execute after timeout, set in config.json. Consider all rollers failed.
		case <-time.After(time.Duration(m.cfg.CollectionTime) * time.Minute):
			if !m.checkAttemptsExceeded(sess.taskID) {
				var success bool
				if message.ProofType(sess.sessionInfos[0].ProofType) == message.ProofTypeBatch {
					success = m.StartBatchProofGenerationSession(nil, sess)
				} else if message.ProofType(sess.sessionInfos[0].ProofType) == message.ProofTypeChunk {
					success = m.StartChunkProofGenerationSession(nil, sess)
				}
				if success {
					m.mu.Lock()
					for _, v := range sess.sessionInfos {
						m.freeTaskIDForRoller(v.RollerPublicKey, v.TaskID)
					}
					m.mu.Unlock()
					log.Info("Retrying session", "session id:", sess.taskID)
					return
				}
			}
			// record failed session.
			errMsg := "proof generation session ended without receiving any valid proofs"
			m.addFailedSession(sess, errMsg)
			log.Warn(errMsg, "session id", sess.taskID)
			// Set status as skipped.
			// Note that this is only a workaround for testnet here.
			// TODO: In real cases we should reset to orm.ProvingTaskUnassigned
			// so as to re-distribute the task in the future
			if message.ProofType(sess.sessionInfos[0].ProofType) == message.ProofTypeChunk {
				if err := m.chunkOrm.UpdateProvingStatus(m.ctx, sess.taskID, types.ProvingTaskFailed); err != nil {
					log.Error("fail to reset chunk task_status as Unassigned", "task id", sess.taskID, "err", err)
				}
			}
			if message.ProofType(sess.sessionInfos[0].ProofType) == message.ProofTypeBatch {
				if err := m.batchOrm.UpdateProvingStatus(m.ctx, sess.taskID, types.ProvingTaskFailed); err != nil {
					log.Error("fail to reset batch task_status as Unassigned", "task id", sess.taskID, "err", err)
				}
			}

			m.mu.Lock()
			for _, v := range sess.sessionInfos {
				m.freeTaskIDForRoller(v.RollerPublicKey, v.TaskID)
			}
			delete(m.sessions, sess.taskID)
			m.mu.Unlock()
			coordinatorSessionsTimeoutTotalCounter.Inc(1)
			return

		//Execute after one of the roller finishes sending proof, return early if all rollers had sent results.
		case ret := <-sess.finishChan:
			m.mu.Lock()
			for idx := range sess.sessionInfos {
				if sess.sessionInfos[idx].RollerPublicKey == ret.pk {
					sess.sessionInfos[idx].ProvingStatus = int16(ret.status)
				}
			}

			if sess.isSessionFailed() {
				if ret.typ == message.ProofTypeChunk {
					if err := m.chunkOrm.UpdateProvingStatus(m.ctx, ret.id, types.ProvingTaskFailed); err != nil {
						log.Error("failed to update chunk proving_status as failed", "msg.ID", ret.id, "error", err)
					}
				}
				if ret.typ == message.ProofTypeBatch {
					if err := m.batchOrm.UpdateProvingStatus(m.ctx, ret.id, types.ProvingTaskFailed); err != nil {
						log.Error("failed to update batch proving_status as failed", "msg.ID", ret.id, "error", err)
					}
				}
				coordinatorSessionsFailedTotalCounter.Inc(1)
			}

			if err := m.sessionInfoOrm.UpdateSessionInfoProvingStatus(m.ctx, ret.typ, ret.id, ret.pk, ret.status); err != nil {
				log.Error("failed to update session info proving status",
					"proof type", ret.typ, "task id", ret.id, "pk", ret.pk, "status", ret.status, "error", err)
			}

			//Check if all rollers have finished their tasks, and rollers with valid results are indexed by public key.
			finished, validRollers := sess.isRollersFinished()

			//When all rollers have finished submitting their tasks, select a winner within rollers with valid proof, and return, terminate the for loop.
			if finished && len(validRollers) > 0 {
				//Select a random index for this slice.
				randIndex := rand.Int63n(int64(len(validRollers)))
				_ = validRollers[randIndex]
				// TODO: reward winner
				for _, sessionInfo := range sess.sessionInfos {
					m.freeTaskIDForRoller(sessionInfo.RollerPublicKey, sessionInfo.TaskID)
					delete(m.sessions, sessionInfo.TaskID)
				}
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
	for _, sessionInfo := range s.sessionInfos {
		if types.RollerProveStatus(sessionInfo.ProvingStatus) == types.RollerProofValid {
			validRollers = append(validRollers, sessionInfo.RollerPublicKey)
			continue
		}

		if types.RollerProveStatus(sessionInfo.ProvingStatus) == types.RollerProofInvalid {
			continue
		}

		// Some rollers are still proving.
		return false, nil
	}
	return true, validRollers
}

func (s *session) isSessionFailed() bool {
	for _, sessionInfo := range s.sessionInfos {
		if types.RollerProveStatus(sessionInfo.ProvingStatus) != types.RollerProofInvalid {
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

// StartChunkProofGenerationSession starts a chunk proof generation session
func (m *Manager) StartChunkProofGenerationSession(task *orm.Chunk, prevSession *session) (success bool) {
	var taskID string
	if task != nil {
		taskID = task.Hash
	} else {
		taskID = prevSession.taskID
	}
	if m.GetNumberOfIdleRollers(message.ProofTypeChunk) == 0 {
		log.Warn("no idle chunk roller when starting proof generation session", "id", taskID)
		return false
	}

	log.Info("start chunk proof generation session", "id", taskID)

	defer func() {
		if !success {
			if task != nil {
				if err := m.chunkOrm.UpdateProvingStatus(m.ctx, taskID, types.ProvingTaskUnassigned); err != nil {
					log.Error("fail to reset task_status as Unassigned", "id", taskID, "err", err)
				}
			} else {
				if err := m.chunkOrm.UpdateProvingStatus(m.ctx, taskID, types.ProvingTaskFailed); err != nil {
					log.Error("fail to reset task_status as Failed", "id", taskID, "err", err)
				}
			}
		}
	}()

	// Get block hashes.
	wrappedBlocks, err := m.l2BlockOrm.GetL2BlocksByChunkHash(m.ctx, taskID)
	if err != nil {
		log.Error(
			"Failed to fetch wrapped blocks",
			"batch hash", taskID,
			"error", err,
		)
		return false
	}
	blockHashes := make([]common.Hash, len(wrappedBlocks))
	for i, wrappedBlock := range wrappedBlocks {
		blockHashes[i] = wrappedBlock.Header.Hash()
	}

	// Dispatch task to chunk rollers.
	var sessionInfos []*orm.SessionInfo
	for i := 0; i < int(m.cfg.RollersPerSession); i++ {
		roller := m.selectRoller(message.ProofTypeChunk)
		if roller == nil {
			log.Info("selectRoller returns nil")
			break
		}
		log.Info("roller is picked", "session id", taskID, "name", roller.Name, "public key", roller.PublicKey)
		// send trace to roller
		if !roller.sendTask(&message.TaskMsg{ID: taskID, Type: message.ProofTypeChunk, BlockHashes: blockHashes}) {
			log.Error("send task failed", "roller name", roller.Name, "public key", roller.PublicKey, "id", taskID)
			continue
		}
		m.updateMetricRollerProofsLastAssignedTimestampGauge(roller.PublicKey)
		tmpSessionInfo := orm.SessionInfo{
			TaskID:          taskID,
			RollerPublicKey: roller.PublicKey,
			ProofType:       int16(message.ProofTypeChunk),
			RollerName:      roller.Name,
			ProvingStatus:   int16(types.RollerAssigned),
			CreatedAt:       time.Now(), // Used in sessionInfos, should be explicitly assigned here.
		}
		// Store session info.
		if err = m.sessionInfoOrm.SetSessionInfo(m.ctx, &tmpSessionInfo); err != nil {
			log.Error("db set session info fail", "session id", taskID, "error", err)
			return false
		}
		sessionInfos = append(sessionInfos, &tmpSessionInfo)
		log.Info("assigned proof to roller", "session id", taskID, "session type", message.ProofTypeChunk, "roller name", roller.Name,
			"roller pk", roller.PublicKey, "proof status", tmpSessionInfo.ProvingStatus)

	}
	// No roller assigned.
	if len(sessionInfos) == 0 {
		log.Error("no roller assigned", "id", taskID, "number of idle chunk rollers", m.GetNumberOfIdleRollers(message.ProofTypeChunk))
		return false
	}

	// Update session proving status as assigned.
	if err = m.chunkOrm.UpdateProvingStatus(m.ctx, taskID, types.ProvingTaskAssigned); err != nil {
		log.Error("failed to update task status", "id", taskID, "err", err)
		return false
	}

	// Create a proof generation session.
	sess := &session{
		taskID:       taskID,
		sessionInfos: sessionInfos,
		finishChan:   make(chan rollerProofStatus, proofAndPkBufferSize),
	}

	m.mu.Lock()
	m.sessions[taskID] = sess
	m.mu.Unlock()
	go m.CollectProofs(sess)

	return true
}

// StartBatchProofGenerationSession starts an batch proof generation.
func (m *Manager) StartBatchProofGenerationSession(task *orm.Batch, prevSession *session) (success bool) {
	var taskID string
	if task != nil {
		taskID = task.Hash
	} else {
		taskID = prevSession.taskID
	}
	if m.GetNumberOfIdleRollers(message.ProofTypeBatch) == 0 {
		log.Warn("no idle common roller when starting proof generation session", "id", taskID)
		return false
	}

	log.Info("start batch proof generation session", "id", taskID)

	defer func() {
		if !success {
			if task != nil {
				if err := m.batchOrm.UpdateProvingStatus(m.ctx, taskID, types.ProvingTaskUnassigned); err != nil {
					log.Error("fail to reset task_status as Unassigned", "id", taskID, "err", err)
				} else if err := m.batchOrm.UpdateProvingStatus(m.ctx, taskID, types.ProvingTaskFailed); err != nil {
					log.Error("fail to reset task_status as Failed", "id", taskID, "err", err)
				}
			}
		}

	}()

	// get chunk proofs from db
	chunkProofs, err := m.chunkOrm.GetProofsByBatchHash(m.ctx, taskID)
	if err != nil {
		log.Error("failed to get chunk proofs for batch task", "session id", taskID, "error", err)
		return false
	}

	// Dispatch task to chunk rollers.
	var sessionInfos []*orm.SessionInfo
	for i := 0; i < int(m.cfg.RollersPerSession); i++ {
		roller := m.selectRoller(message.ProofTypeBatch)
		if roller == nil {
			log.Info("selectRoller returns nil")
			break
		}
		log.Info("roller is picked", "session id", taskID, "name", roller.Name, "type", roller.Type, "public key", roller.PublicKey)
		// send trace to roller
		if !roller.sendTask(&message.TaskMsg{
			ID:        taskID,
			Type:      message.ProofTypeBatch,
			SubProofs: chunkProofs,
		}) {
			log.Error("send task failed", "roller name", roller.Name, "public key", roller.PublicKey, "id", taskID)
			continue
		}

		tmpSessionInfo := orm.SessionInfo{
			TaskID:          taskID,
			RollerPublicKey: roller.PublicKey,
			ProofType:       int16(message.ProofTypeBatch),
			RollerName:      roller.Name,
			ProvingStatus:   int16(types.RollerAssigned),
			CreatedAt:       time.Now(), // Used in sessionInfos, should be explicitly assigned here.
		}
		// Store session info.
		if err = m.sessionInfoOrm.SetSessionInfo(context.Background(), &tmpSessionInfo); err != nil {
			log.Error("db set session info fail", "session id", taskID, "error", err)
			return false
		}

		m.updateMetricRollerProofsLastAssignedTimestampGauge(roller.PublicKey)
		sessionInfos = append(sessionInfos, &tmpSessionInfo)
		log.Info("assigned proof to roller", "session id", taskID, "session type", message.ProofTypeBatch, "roller name", roller.Name,
			"roller pk", roller.PublicKey, "proof status", tmpSessionInfo.ProvingStatus)
	}
	// No roller assigned.
	if len(sessionInfos) == 0 {
		log.Error("no roller assigned", "id", taskID, "number of idle batch rollers", m.GetNumberOfIdleRollers(message.ProofTypeBatch))
		return false
	}

	// Update session proving status as assigned.
	if err = m.batchOrm.UpdateProvingStatus(m.ctx, taskID, types.ProvingTaskAssigned); err != nil {
		log.Error("failed to update task status", "id", taskID, "err", err)
		return false
	}

	// Create a proof generation session.
	sess := &session{
		taskID:       taskID,
		sessionInfos: sessionInfos,
		finishChan:   make(chan rollerProofStatus, proofAndPkBufferSize),
	}

	m.mu.Lock()
	m.sessions[taskID] = sess
	m.mu.Unlock()
	go m.CollectProofs(sess)

	return true
}

func (m *Manager) addFailedSession(sess *session, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedSessionInfos[sess.taskID] = newSessionInfo(sess, types.ProvingTaskFailed, errMsg, true)
}

// VerifyToken verifies pukey for token and expiration time
func (m *Manager) VerifyToken(authMsg *message.AuthMsg) (bool, error) {
	pubkey, _ := authMsg.PublicKey()
	// GetValue returns nil if value is expired
	if token, ok := m.tokenCache.Get(pubkey); !ok || token != authMsg.Identity.Token {
		return false, fmt.Errorf("failed to find corresponding token. roller name: %s. roller pk: %s", authMsg.Identity.Name, pubkey)
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
	result := <-verifyResultChan
	return result.result, result.err
}

type verifyResult struct {
	result bool
	err    error
}
