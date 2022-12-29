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
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/message"
	"scroll-tech/common/viper"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/coordinator/verifier"
)

const (
	proofAndPkBufferSize = 10
)

type rollerProofStatus struct {
	id     string
	pk     string
	status orm.RollerProveStatus
}

// Contains all the information on an ongoing proof generation session.
type session struct {
	info *orm.SessionInfo
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

	// The indicator whether the backend is running or not.
	running int32

	vp *viper.Viper

	// A mutex guarding the boolean below.
	mu sync.RWMutex
	// A map containing all active proof generation sessions.
	sessions map[string]*session
	// A map containing proof failed or verify failed proof.
	rollerPool cmap.ConcurrentMap

	// TODO: once put into use, should add to graceful restart.
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
}

// New returns a new instance of Manager. The instance will be not fully prepared,
// and still needs to be finalized and ran by calling `manager.Start`.
func New(ctx context.Context, vp *viper.Viper, orm database.OrmFactory, client *ethclient.Client) (*Manager, error) {
	verifier, err := verifier.NewVerifier(vp.Sub("verifier"))
	if err != nil {
		return nil, err
	}

	log.Info("Start coordinator successfully.")
	return &Manager{
		ctx:                ctx,
		rollerPool:         cmap.New(),
		sessions:           make(map[string]*session),
		failedSessionInfos: make(map[string]*SessionInfo),
		verifier:           verifier,
		orm:                orm,
		vp:                 vp,
		Client:             client,
		tokenCache:         cache.New(time.Duration(vp.GetInt("token_time_to_live"))*time.Second, 1*time.Hour),
	}, nil
}

// Start the Manager module.
func (m *Manager) Start() error {
	if m.isRunning() {
		return nil
	}

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

	atomic.StoreInt32(&m.running, 0)
}

// isRunning returns an indicator whether manager is running or not.
func (m *Manager) isRunning() bool {
	return atomic.LoadInt32(&m.running) == 1
}

// Loop keeps the manager running.
func (m *Manager) Loop() {
	var (
		tick  = time.NewTicker(time.Second * 3)
		tasks []*orm.BlockBatch
	)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if len(tasks) == 0 && m.orm != nil {
				var err error
				// TODO: add cache
				if tasks, err = m.orm.GetBlockBatches(
					map[string]interface{}{"proving_status": orm.ProvingTaskUnassigned},
					fmt.Sprintf(
						"ORDER BY index %s LIMIT %d;",
						m.vp.GetString("order_session"),
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
	if ids, err := m.orm.GetAssignedBatchIDs(); err != nil {
		log.Error("failed to get assigned batch ids from db", "error", err)
	} else if prevSessions, err := m.orm.GetSessionInfosByIDs(ids); err != nil {
		log.Error("failed to recover roller session info from db", "error", err)
	} else {
		for _, v := range prevSessions {
			sess := &session{
				info:       v,
				finishChan: make(chan rollerProofStatus, proofAndPkBufferSize),
			}
			m.sessions[sess.info.ID] = sess
			log.Info("Coordinator restart reload sessions", "ID", sess.info.ID, "sess", sess.info)
			go m.CollectProofs(sess.info.ID, sess)
		}
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
		return fmt.Errorf("proof generation session for id %v does not exist", msg.ID)
	}
	proofTimeSec := uint64(time.Since(time.Unix(sess.info.StartTimestamp, 0)).Seconds())

	// Ensure this roller is eligible to participate in the session.
	if roller, ok := sess.info.Rollers[pk]; !ok {
		return fmt.Errorf("roller %s is not eligible to partake in proof session %v", pk, msg.ID)
	} else if roller.Status == orm.RollerProofValid {
		// In order to prevent DoS attacks, it is forbidden to repeatedly submit valid proofs.
		// TODO: Defend invalid proof resubmissions by one of the following two methods:
		// (i) slash the roller for each submission of invalid proof
		// (ii) set the maximum failure retry times
		log.Warn("roller has already submitted valid proof in proof session", "roller", pk, "proof id", msg.ID)
		return nil
	}
	log.Info("Received zk proof", "proof id", msg.ID)

	defer func() {
		// TODO: maybe we should use db tx for the whole process?
		// Roll back current proof's status.
		if dbErr != nil {
			if err := m.orm.UpdateProvingStatus(msg.ID, orm.ProvingTaskUnassigned); err != nil {
				log.Error("fail to reset task status as Unassigned", "msg.ID", msg.ID)
			}
		}
		// set proof status
		status := orm.RollerProofInvalid
		if success && dbErr == nil {
			status = orm.RollerProofValid
		}
		// notify the session that the roller finishes the proving process
		sess.finishChan <- rollerProofStatus{msg.ID, pk, status}
	}()

	if msg.Status != message.StatusOk {
		log.Error("Roller failed to generate proof", "msg.ID", msg.ID, "error", msg.Error)
		if dbErr = m.orm.UpdateProvingStatus(msg.ID, orm.ProvingTaskFailed); dbErr != nil {
			log.Error("failed to update task status as failed", "error", dbErr)
		}
		// record the failed session.
		m.addFailedSession(sess, msg.Error)
		return nil
	}

	// store proof content
	if dbErr = m.orm.UpdateProofByID(m.ctx, msg.ID, msg.Proof.Proof, msg.Proof.FinalPair, proofTimeSec); dbErr != nil {
		log.Error("failed to store proof into db", "error", dbErr)
		return dbErr
	}
	if dbErr = m.orm.UpdateProvingStatus(msg.ID, orm.ProvingTaskProved); dbErr != nil {
		log.Error("failed to update task status as proved", "error", dbErr)
		return dbErr
	}

	var err error
	tasks, err := m.orm.GetBlockBatches(map[string]interface{}{"id": msg.ID})
	if len(tasks) == 0 {
		if err != nil {
			log.Error("failed to get tasks", "error", err)
		}
		return err
	}

	success, err = m.verifier.VerifyProof(msg.Proof)
	if err != nil {
		// record failed session.
		m.addFailedSession(sess, err.Error())
		// TODO: this is only a temp workaround for testnet, we should return err in real cases
		success = false
		log.Error("Failed to verify zk proof", "proof id", msg.ID, "error", err)
		// TODO: Roller needs to be slashed if proof is invalid.
	} else {
		log.Info("Verify zk proof successfully", "verification result", success, "proof id", msg.ID)
	}

	var status orm.ProvingStatus
	if success {
		status = orm.ProvingTaskVerified
	} else {
		// Set status as skipped if verification fails.
		// Note that this is only a workaround for testnet here.
		// TODO: In real cases we should reset to orm.ProvingTaskUnassigned
		// so as to re-distribute the task in the future
		status = orm.ProvingTaskFailed
	}
	if dbErr = m.orm.UpdateProvingStatus(msg.ID, status); dbErr != nil {
		log.Error("failed to update proving_status", "msg.ID", msg.ID, "status", status, "error", dbErr)
	}

	return dbErr
}

// CollectProofs collects proofs corresponding to a proof generation session.
func (m *Manager) CollectProofs(id string, sess *session) {
	timer := time.NewTimer(time.Duration(m.vp.GetInt("collection_time")) * time.Minute)

	for {
		select {
		case <-timer.C:
			m.mu.Lock()

			// Ensure proper clean-up of resources.
			defer func() {
				delete(m.sessions, id)
				m.mu.Unlock()
			}()

			// Pick a random winner.
			// First, round up the keys that actually sent in a valid proof.
			var participatingRollers []string
			for pk, roller := range sess.info.Rollers {
				if roller.Status == orm.RollerProofValid {
					participatingRollers = append(participatingRollers, pk)
				}
			}
			// Ensure we got at least one proof before selecting a winner.
			if len(participatingRollers) == 0 {
				// record failed session.
				errMsg := "proof generation session ended without receiving any valid proofs"
				m.addFailedSession(sess, errMsg)
				log.Warn(errMsg, "session id", id)
				// Set status as skipped.
				// Note that this is only a workaround for testnet here.
				// TODO: In real cases we should reset to orm.ProvingTaskUnassigned
				// so as to re-distribute the task in the future
				if err := m.orm.UpdateProvingStatus(id, orm.ProvingTaskFailed); err != nil {
					log.Error("fail to reset task_status as Unassigned", "id", id, "err", err)
				}
				return
			}

			// Now, select a random index for this slice.
			randIndex := mathrand.Intn(len(participatingRollers))
			_ = participatingRollers[randIndex]
			// TODO: reward winner
			return
		case ret := <-sess.finishChan:
			m.mu.Lock()
			sess.info.Rollers[ret.pk].Status = ret.status
			m.mu.Unlock()
			if err := m.orm.SetSessionInfo(sess.info); err != nil {
				log.Error("db set session info fail", "pk", ret.pk, "error", err)
			}
		}
	}
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
func (m *Manager) StartProofGenerationSession(task *orm.BlockBatch) (success bool) {
	roller := m.selectRoller()
	if roller == nil {
		return false
	}
	log.Info("start proof generation session", "id", task.ID)

	defer func() {
		if !success {
			if err := m.orm.UpdateProvingStatus(task.ID, orm.ProvingTaskUnassigned); err != nil {
				log.Error("fail to reset task_status as Unassigned", "id", task.ID, "err", err)
			}
		}
	}()
	if err := m.orm.UpdateProvingStatus(task.ID, orm.ProvingTaskAssigned); err != nil {
		log.Error("failed to update task status", "id", task.ID, "err", err)
		return false
	}

	blockInfos, err := m.orm.GetBlockInfos(map[string]interface{}{"batch_id": task.ID})
	if err != nil {
		log.Error(
			"could not GetBlockInfos",
			"batch_id", task.ID,
			"error", err,
		)
		return false
	}

	traces := make([]*types.BlockTrace, len(blockInfos))
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

	log.Info("roller is picked", "name", roller.Name, "public_key", roller.PublicKey)

	// send trace to roller
	if !roller.sendTask(task.ID, traces) {
		log.Error("send task failed", "roller name", roller.Name, "id", task.ID)
		return false
	}

	pk := roller.PublicKey
	// Create a proof generation session.
	s := &session{
		info: &orm.SessionInfo{
			ID: task.ID,
			Rollers: map[string]*orm.RollerStatus{
				pk: {
					PublicKey: pk,
					Name:      roller.Name,
					Status:    orm.RollerAssigned,
				},
			},
			StartTimestamp: time.Now().Unix(),
		},
		finishChan: make(chan rollerProofStatus, proofAndPkBufferSize),
	}

	// Store session info.
	if err = m.orm.SetSessionInfo(s.info); err != nil {
		log.Error("db set session info fail", "pk", pk, "error", err)
		return false
	}

	m.mu.Lock()
	m.sessions[task.ID] = s
	m.mu.Unlock()
	go m.CollectProofs(task.ID, s)

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
			if pk == hexPk && roller.Status == orm.RollerAssigned {
				return false
			}
		}
	}

	return true
}

func (m *Manager) addFailedSession(sess *session, errMsg string) {
	m.failedSessionInfos[sess.info.ID] = newSessionInfo(sess, orm.ProvingTaskFailed, errMsg, true)
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
