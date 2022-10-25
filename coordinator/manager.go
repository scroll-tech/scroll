package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	mathrand "math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/message"
	"scroll-tech/database/orm"

	"scroll-tech/coordinator/config"
	"scroll-tech/coordinator/verifier"
)

const (
	proofAndPkBufferSize = 10
)

// Contains all the information on an ongoing proof generation session.
type session struct {
	// session id
	id uint64
	// A list of all participating rollers and if they finished proof generation for this session.
	// The map key is a hexadecimal encoding of the roller public key, as byte slices
	// can not be compared explicitly.
	rollers      map[string]bool
	roller_names map[string]string
	// session start time
	startTime time.Time
	// finish channel is used to pass the public key of the rollers who finished proving process.
	finishChan chan string
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
	// The websocket server which holds the connections with active rollers.
	server *server

	// A mutex guarding the boolean below.
	mu sync.RWMutex
	// A map containing all active proof generation sessions.
	sessions map[uint64]session
	// A map containing proof failed or verify failed proof.
	failedSessionInfos map[uint64]SessionInfo

	// A direct connection to the Halo2 verifier, used to verify
	// incoming proofs.
	verifier *verifier.Verifier

	// db interface
	orm orm.ProveTaskOrm
}

// New returns a new instance of Manager. The instance will be not fully prepared,
// and still needs to be finalized and ran by calling `manager.Start`.
func New(ctx context.Context, cfg *config.RollerManagerConfig, orm orm.ProveTaskOrm) (*Manager, error) {
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
		server:             newServer(cfg.Endpoint),
		sessions:           make(map[uint64]session),
		failedSessionInfos: make(map[uint64]SessionInfo),
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
		tasks, err := m.orm.GetProveTasks(map[string]interface{}{"status": orm.TaskAssigned})
		if err == nil {
			for _, task := range tasks {
				if err := m.orm.UpdateTaskStatus(task.ID, orm.TaskUnassigned); err != nil {
					log.Error("fail to reset task as Unassigned")
				}
			}
		} else {
			log.Error("fail to fetch assigned tasks")
		}
	}

	if err := m.server.start(); err != nil {
		return err
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

	// Stop accepting connections
	if err := m.server.stop(); err != nil {
		log.Error("Server shutdown failed", "error", err)
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
		tasks []*orm.ProveTask
	)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if len(tasks) == 0 && m.orm != nil {
				var err error
				numIdleRollers := m.GetNumberOfIdleRollers()
				// TODO: add cache
				if tasks, err = m.orm.GetProveTasks(
					map[string]interface{}{"status": orm.TaskUnassigned},
					fmt.Sprintf(
						"ORDER BY id %s LIMIT %d;",
						m.cfg.OrderSession,
						numIdleRollers,
					),
				); err != nil {
					log.Error("failed to GetProveTasks", "error", err)
					continue
				}
			}
			// Select roller and send message
			for len(tasks) > 0 && m.StartProofGenerationSession(tasks[0]) {
				tasks = tasks[1:]
			}
		case msg := <-m.server.msgChan:
			if err := m.HandleMessage(msg.pk, msg.message); err != nil {
				log.Error(
					"could not handle message",
					"error", err,
				)
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

// HandleMessage handle a message from a roller.
func (m *Manager) HandleMessage(pk string, payload []byte) error {
	// Recover message
	msg := &message.Msg{}
	if err := json.Unmarshal(payload, msg); err != nil {
		return err
	}

	switch msg.Type {
	case message.ErrorMsgType:
		// Just log it for now.
		log.Error("error message received from roller", "message", msg)
		// TODO: handle in m.failedSessionInfos
		return nil
	case message.RegisterMsgType:
		// We shouldn't get this message, as the sequencer should handle registering at the start
		// of the connection.
		return errors.New("attempted handshake at the wrong time")
	case message.TaskMsgType:
		// We shouldn't get this message, as the sequencer should always be the one to send it
		return errors.New("received illegal message")
	case message.ProofMsgType:
		return m.HandleZkProof(pk, msg.Payload)
	default:
		return fmt.Errorf("unrecognized message type %v", msg.Type)
	}
}

// HandleZkProof handle a ZkProof submitted from a roller.
// For now only proving/verifying error will lead to setting status as skipped.
// db/unmarshal errors will not because they are errors on the business logic side.
func (m *Manager) HandleZkProof(pk string, payload []byte) error {
	var dbErr error

	msg := &message.ProofMsg{}
	if err := json.Unmarshal(payload, msg); err != nil {
		return err
	}

	// Assess if the proof generation session for the given ID is still active.
	// We hold the read lock until the end of the function so that there is no
	// potential race for channel deletion.
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[msg.ID]
	if !ok {
		return fmt.Errorf("proof generation session for id %v does not exist", msg.ID)
	}
	proofTimeSec := uint64(time.Since(s.startTime).Seconds())

	// Ensure this roller is eligible to participate in the session.
	if _, ok = s.rollers[pk]; !ok {
		return fmt.Errorf("roller %s is not eligible to partake in proof session %v", pk, msg.ID)
	}
	log.Info("Received zk proof", "proof id", msg.ID)

	defer func() {
		// notify the session that the roller finishes the proving process
		s.finishChan <- pk
		// TODO: maybe we should use db tx for the whole process?
		// Roll back current proof's status.
		if dbErr != nil {
			if err := m.orm.UpdateTaskStatus(msg.ID, orm.TaskUnassigned); err != nil {
				log.Error("fail to reset task_status as Unassigned", "msg.ID", msg.ID)
			}
		}
	}()

	if msg.Status != message.StatusOk {
		log.Error("Roller failed to generate proof", "msg.ID", msg.ID, "error", msg.Error)
		if dbErr = m.orm.UpdateTaskStatus(msg.ID, orm.TaskFailed); dbErr != nil {
			log.Error("failed to update task status as failed", "error", dbErr)
		}
		// record the failed session.
		m.addFailedSession(&s, msg.Error)
		return nil
	}

	// store proof content
	if dbErr = m.orm.UpdateProofByID(m.ctx, msg.ID, msg.Proof.Proof, msg.Proof.FinalPair, proofTimeSec); dbErr != nil {
		log.Error("failed to store proof into db", "error", dbErr)
		return dbErr
	}
	if dbErr = m.orm.UpdateTaskStatus(msg.ID, orm.TaskProved); dbErr != nil {
		log.Error("failed to update task status as proved", "error", dbErr)
		return dbErr
	}

	var success bool
	if m.verifier != nil {
		// TODO: fix
		var err error
		// tasks, err := m.orm.GetProveTasks(map[string]interface{}{"number": msg.ID})
		// if len(tasks) == 0 {
		// 	if err != nil {
		// 		log.Error("failed to get tasks", "error", err)
		// 	}
		// 	return err
		// }

		success, err = m.verifier.VerifyProof(nil, msg.Proof)
		if err != nil {
			// record failed session.
			m.addFailedSession(&s, err.Error())
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

	var status orm.TaskStatus
	if success {
		status = orm.TaskVerified
	} else {
		// Set status as skipped if verification fails.
		// Note that this is only a workaround for testnet here.
		// TODO: In real cases we should reset to orm.TaskUnassigned
		// so as to re-distribute the task in the future
		status = orm.TaskFailed
	}
	if dbErr = m.orm.UpdateTaskStatus(msg.ID, status); dbErr != nil {
		log.Error("failed to update blockResult status", "status", status, "error", dbErr)
	}

	return dbErr
}

// CollectProofs collects proofs corresponding to a proof generation session.
func (m *Manager) CollectProofs(id uint64, s session) {
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
			// First, round up the keys that actually sent in a proof.
			var participatingRollers []string
			for pk, finished := range s.rollers {
				if finished {
					participatingRollers = append(participatingRollers, pk)
				}
			}
			// Ensure we got at least one proof before selecting a winner.
			if len(participatingRollers) == 0 {
				// record failed session.
				errMsg := "proof generation session ended without receiving any proofs"
				m.addFailedSession(&s, errMsg)
				log.Warn(errMsg, "session id", id)
				// Set status as skipped.
				// Note that this is only a workaround for testnet here.
				// TODO: In real cases we should reset to orm.TaskUnassigned
				// so as to re-distribute the task in the future
				if err := m.orm.UpdateTaskStatus(id, orm.TaskFailed); err != nil {
					log.Error("fail to reset task_status as Unassigned", "id", id)
				}
				return
			}

			// Now, select a random index for this slice.
			randIndex := mathrand.Intn(len(participatingRollers))
			_ = participatingRollers[randIndex]
			// TODO: reward winner
			return
		case pk := <-s.finishChan:
			m.mu.Lock()
			s.rollers[pk] = true
			m.mu.Unlock()
		}
	}
}

// GetRollerChan returns the channel in which newly created wrapped roller connections are sent.
func (m *Manager) GetRollerChan() chan *Roller {
	return m.server.rollerChan
}

// APIs collect API services.
func (m *Manager) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "roller",
			Public:    true,
			Service:   RollerDebugAPI(m),
		},
	}
}

// StartProofGenerationSession starts a proof generation session
func (m *Manager) StartProofGenerationSession(task *orm.ProveTask) bool {
	roller := m.SelectRoller()
	if roller == nil || roller.isClosed() {
		return false
	}

	id := task.ID
	log.Info("start proof generation session", "id", id)

	var dbErr error
	defer func() {
		if dbErr != nil {
			if err := m.orm.UpdateTaskStatus(id, orm.TaskUnassigned); err != nil {
				log.Error("fail to reset task_status as Unassigned", "id", id)
			}
		}
	}()

	pk := roller.AuthMsg.Identity.PublicKey
	log.Info("roller is picked", "name", roller.AuthMsg.Identity.Name, "public_key", pk)

	msg, err := createTaskMsg(task)
	if err != nil {
		log.Error(
			"could not create block traces message",
			"error", err,
		)
		return false
	}
	// TODO: use some compression?
	if err := roller.sendMessage(msg); err != nil {
		log.Error(
			"could not send traces message to roller",
			"error", err,
		)
		return false
	}

	s := session{
		id: id,
		rollers: map[string]bool{
			pk: false,
		},
		roller_names: map[string]string{
			pk: roller.AuthMsg.Identity.Name,
		},
		startTime:  time.Now(),
		finishChan: make(chan string, proofAndPkBufferSize),
	}

	// Create a proof generation session.
	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()

	dbErr = m.orm.UpdateTaskStatus(id, orm.TaskAssigned)
	go m.CollectProofs(id, s)

	return true
}

// SelectRoller randomly get one idle roller.
func (m *Manager) SelectRoller() *Roller {
	allRollers := m.server.conns.getAll()
	for len(allRollers) > 0 {
		idx := mathrand.Intn(len(allRollers))
		conn := allRollers[idx]
		pk := conn.AuthMsg.Identity.PublicKey
		if conn.isClosed() {
			log.Debug("roller is closed", "public_key", pk)
			// Delete closed connection.
			m.server.conns.delete(conn)
			// Delete the offline roller.
			allRollers[idx], allRollers = allRollers[0], allRollers[1:]
			continue
		}
		// Ensure the roller is not currently working on another session.
		if !m.IsRollerIdle(pk) {
			log.Debug("roller is busy", "public_key", pk)
			// Delete the busy roller.
			allRollers[idx], allRollers = allRollers[0], allRollers[1:]
			continue
		}
		return conn
	}
	return nil
}

// IsRollerIdle determines whether this roller is idle.
func (m *Manager) IsRollerIdle(hexPk string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// We need to iterate over all sessions because finished sessions will be deleted until the
	// timeout. So a busy roller could be marked as idle in a finished session.
	for _, sess := range m.sessions {
		for pk, finished := range sess.rollers {
			if pk == hexPk && !finished {
				return false
			}
		}
	}

	return true
}

// GetNumberOfIdleRollers returns the number of idle rollers in maintain list
func (m *Manager) GetNumberOfIdleRollers() int {
	cnt := 0
	// m.server.conns doesn't have any lock
	for _, roller := range m.server.conns.getAll() {
		if m.IsRollerIdle(roller.AuthMsg.Identity.PublicKey) {
			cnt++
		}
	}
	return cnt
}

// TODO: implement this
func createTaskMsg(task *orm.ProveTask) (message.Msg, error) {
	idAndTraces := message.Task{
		ID:     task.ID,
		Traces: nil,
	}

	payload, err := json.Marshal(idAndTraces)
	if err != nil {
		return message.Msg{}, err
	}

	return message.Msg{
		Type:    message.TaskMsgType,
		Payload: payload,
	}, nil
}

func (m *Manager) addFailedSession(s *session, errMsg string) {
	m.failedSessionInfos[s.id] = *newSessionInfo(s, orm.TaskFailed, errMsg, true)
}
