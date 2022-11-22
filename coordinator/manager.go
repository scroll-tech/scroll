package coordinator

import (
	"context"
	"fmt"
	mathrand "math/rand"
	"sync"
	"sync/atomic"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/message"
	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/coordinator/config"
	"scroll-tech/coordinator/verifier"
)

const (
	proofAndPkBufferSize = 10
)

type rollerStatus int32

const (
	rollerAssigned rollerStatus = iota
	rollerProofValid
	rollerProofInvalid
)

type rollerProofStatus struct {
	pk     string
	status rollerStatus
}

// Contains all the information on an ongoing proof generation session.
type session struct {
	// session id
	id string
	// A list of all participating rollers and if they finished proof generation for this session.
	// The map key is a hexadecimal encoding of the roller public key, as byte slices
	// can not be compared explicitly.
	rollers     map[string]rollerStatus
	rollerNames map[string]string
	// session start time
	startTime time.Time
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

	rollerPool         cmap.ConcurrentMap
	failedSessionInfos map[string]*SessionInfo

	// A direct connection to the Halo2 verifier, used to verify
	// incoming proofs.
	verifier *verifier.Verifier

	// db interface
	orm database.OrmFactory
}

// New returns a new instance of Manager. The instance will be not fully prepared,
// and still needs to be finalized and ran by calling `manager.Start`.
func New(ctx context.Context, cfg *config.RollerManagerConfig, orm database.OrmFactory) (*Manager, error) {
	var v *verifier.Verifier
	if cfg.VerifierEndpoint != "" {
		var err error
		v, err = verifier.NewVerifier(cfg.VerifierEndpoint)
		if err != nil {
			return nil, err
		}
	}

	log.Info("Start rollerManager successfully.")

	return &Manager{
		ctx:                ctx,
		cfg:                cfg,
		rollerPool:         cmap.New(),
		sessions:           make(map[string]*session),
		failedSessionInfos: make(map[string]*SessionInfo),
		verifier:           v,
		orm:                orm,
	}, nil
}

// Start the Manager module.
func (m *Manager) Start() error {
	if m.isRunning() {
		return nil
	}

	// m.orm may be nil in scroll tests
	if m.orm != nil {
		// clean up assigned but not submitted task
		if err := m.orm.ResetProvingStatusFor(orm.ProvingTaskAssigned); err != nil {
			log.Error("fail to reset assigned tasks as unassigned")
		}
	}

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

// HandleZkProof handle a ZkProof submitted from a roller.
// For now only proving/verifying error will lead to setting status as skipped.
// db/unmarshal errors will not because they are errors on the business logic side.
func (m *Manager) handleZkProof(pk string, msg *message.ProofMsg) error {
	var dbErr error
	var success bool

	// Assess if the proof generation session for the given ID is still active.
	// We hold the read lock until the end of the function so that there is no
	// potential race for channel deletion.
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[msg.ID]
	if !ok {
		return fmt.Errorf("proof generation session for id %v does not existID", msg.ID)
	}
	proofTimeSec := uint64(time.Since(s.startTime).Seconds())

	// Ensure this roller is eligible to participate in the session.
	if status, ok := s.rollers[pk]; !ok {
		return fmt.Errorf("roller %s is not eligible to partake in proof session %v", pk, msg.ID)
	} else if status == rollerProofValid {
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
		var status rollerStatus
		if success && dbErr == nil {
			status = rollerProofValid
		} else {
			status = rollerProofInvalid
		}
		// notify the session that the roller finishes the proving process
		s.finishChan <- rollerProofStatus{pk, status}
	}()

	if msg.Status != message.StatusOk {
		log.Error("Roller failed to generate proof", "msg.ID", msg.ID, "error", msg.Error)
		if dbErr = m.orm.UpdateProvingStatus(msg.ID, orm.ProvingTaskFailed); dbErr != nil {
			log.Error("failed to update task status as failed", "error", dbErr)
		}
		// record the failed session.
		m.addFailedSession(s, msg.Error)
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

	if m.verifier != nil {
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
			m.addFailedSession(s, err.Error())
			// TODO: this is only a temp workaround for testnet, we should return err in real cases
			success = false
			log.Error("Failed to verify zk proof", "proof id", msg.ID, "error", err)
			// TODO: Roller needs to be slashed if proof is invalid.
		} else {
			log.Info("Verify zk proof successfully", "verification result", success, "proof id", msg.ID)
		}
	} else {
		success = true
		log.Info("Verifier disabled, VerifyProof skipped")
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
func (m *Manager) CollectProofs(id string, s *session) {
	timer := time.NewTimer(time.Duration(m.cfg.CollectionTime) * time.Minute)

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
			for pk, status := range s.rollers {
				if status == rollerProofValid {
					participatingRollers = append(participatingRollers, pk)
				}
			}
			// Ensure we got at least one proof before selecting a winner.
			if len(participatingRollers) == 0 {
				// record failed session.
				errMsg := "proof generation session ended without receiving any valid proofs"
				m.addFailedSession(s, errMsg)
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
		case ret := <-s.finishChan:
			m.mu.Lock()
			s.rollers[ret.pk] = ret.status
			m.mu.Unlock()
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
func (m *Manager) StartProofGenerationSession(task *orm.BlockBatch) bool {
	roller := m.selectRoller()
	if roller == nil {
		return false
	}

	log.Info("start proof generation session", "id", task.ID)

	var dbErr error
	defer func() {
		if dbErr != nil {
			log.Error("StartProofGenerationSession", "dbErr", dbErr)
			if err := m.orm.UpdateProvingStatus(task.ID, orm.ProvingTaskUnassigned); err != nil {
				log.Error("fail to reset task_status as Unassigned", "id", task.ID, "dbErr", dbErr, "err", err)
			}
		}
	}()

	traces, err := m.orm.GetBlockTraces(map[string]interface{}{"batch_id": task.ID})
	if err != nil {
		log.Error(
			"could not GetBlockTraces",
			"batch_id", task.ID,
			"error", err,
		)
		return false
	}

	log.Info("roller is picked", "name", roller.Name, "public_key", roller.PublicKey)
	// send trace to roller
	roller.sendMsg(task.ID, traces)

	pk := roller.PublicKey
	s := &session{
		id: task.ID,
		rollers: map[string]rollerStatus{
			pk: rollerAssigned,
		},
		rollerNames: map[string]string{
			pk: roller.Name,
		},
		startTime:  time.Now(),
		finishChan: make(chan rollerProofStatus, proofAndPkBufferSize),
	}

	// Create a proof generation session.
	m.mu.Lock()
	m.sessions[task.ID] = s
	m.mu.Unlock()

	dbErr = m.orm.UpdateProvingStatus(task.ID, orm.ProvingTaskAssigned)
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
		for pk, status := range sess.rollers {
			if pk == hexPk && status == rollerAssigned {
				return false
			}
		}
	}

	return true
}

func (m *Manager) addFailedSession(s *session, errMsg string) {
	m.failedSessionInfos[s.id] = newSessionInfo(s, orm.ProvingTaskFailed, errMsg, true)
}
