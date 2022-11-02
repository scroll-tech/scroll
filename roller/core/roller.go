package core

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/scroll-tech/go-ethereum/accounts/keystore"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/message"
	"scroll-tech/common/version"

	"scroll-tech/roller/config"
	"scroll-tech/roller/core/prover"
	"scroll-tech/roller/store"
)

// ZK_VERSION is commit-id of prover/rust/cargo.lock/common-rs
var (
	ZK_VERSION string
	Version    = fmt.Sprintf("%s-%s", version.Version, ZK_VERSION)
)

var (
	writeWait = time.Second + readWait
	// consider ping message
	readWait = time.Minute * 30
	// retry connecting to coordinator
	retryWait = time.Second * 10
	// net normal close
	errNormalClose = errors.New("use of closed network connection")
)

// Roller contains websocket conn to coordinator, Stack, unix-socket to ipc-prover.
type Roller struct {
	cfg    *config.Config
	conn   *websocket.Conn
	stack  *store.Stack
	prover *prover.Prover

	isClosed int64
	stopChan chan struct{}
}

// NewRoller new a Roller object.
func NewRoller(cfg *config.Config) (*Roller, error) {
	// Get stack db handler
	stackDb, err := store.NewStack(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	// Create prover instance
	log.Info("init prover")
	pver, err := prover.NewProver(cfg.Prover)
	if err != nil {
		return nil, err
	}
	log.Info("init prover successfully!")

	conn, _, err := websocket.DefaultDialer.Dial(cfg.CoordinatorURL, nil)
	if err != nil {
		return nil, err
	}

	return &Roller{
		cfg:      cfg,
		conn:     conn,
		stack:    stackDb,
		prover:   pver,
		stopChan: make(chan struct{}),
	}, nil
}

// Run runs Roller.
func (r *Roller) Run() error {
	log.Info("start to register to coordinator")
	if err := r.Register(); err != nil {
		log.Crit("register to coordinator failed", "error", err)
	}
	log.Info("register to coordinator successfully!")
	go func() {
		r.HandleCoordinator()
		r.Close()
	}()

	return r.ProveLoop()
}

// Register registers Roller to the coordinator through Websocket.
func (r *Roller) Register() error {
	priv, err := r.loadOrCreateKey()
	if err != nil {
		return err
	}
	authMsg := &message.AuthMessage{
		Identity: message.Identity{
			Name:      r.cfg.RollerName,
			Timestamp: time.Now().UnixMilli(),
			PublicKey: common.Bytes2Hex(crypto.FromECDSAPub(&priv.PublicKey)),
			Version:   Version,
		},
		Signature: "",
	}

	// Sign auth message
	if err = authMsg.Sign(priv); err != nil {
		return fmt.Errorf("sign auth message failed %v", err)
	}

	return r.sendMessage(message.RegisterMsgType, authMsg)
}

// HandleCoordinator accepts block-traces from coordinator through the Websocket and store it into Stack.
func (r *Roller) HandleCoordinator() {
	for {
		select {
		case <-r.stopChan:
			return
		default:
			_ = r.conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = r.conn.SetReadDeadline(time.Now().Add(readWait))
			if err := r.handMessage(); err != nil && !strings.Contains(err.Error(), errNormalClose.Error()) {
				log.Error("handle coordinator failed", "error", err)
				r.mustRetryCoordinator()
				continue
			}
		}
	}
}

func (r *Roller) mustRetryCoordinator() {
	for {
		log.Info("retry to connect to coordinator...")
		conn, _, err := websocket.DefaultDialer.Dial(r.cfg.CoordinatorURL, nil)
		if err != nil {
			log.Error("failed to connect coordinator: ", "error", err)
			time.Sleep(retryWait)
		} else {
			r.conn = conn
			log.Info("re-connect to coordinator successfully!")
			break
		}
	}
	for {
		log.Info("retry to register to coordinator...")
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
func (r *Roller) ProveLoop() (err error) {
	for {
		select {
		case <-r.stopChan:
			return nil
		default:
			_ = r.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err = r.prove(); err != nil {
				if errors.Is(err, store.ErrEmpty) {
					log.Debug("get empty trace", "error", err)
					time.Sleep(time.Second * 3)
					continue
				}
				if strings.Contains(err.Error(), errNormalClose.Error()) {
					return nil
				}
				log.Error("prove failed", "error", err)
			}
		}
	}
}

func (r *Roller) sendMessage(msgType message.MsgType, payload interface{}) error {
	msgByt, err := MakeMsgByt(msgType, payload)
	if err != nil {
		return err
	}
	return r.conn.WriteMessage(websocket.BinaryMessage, msgByt)
}

func (r *Roller) handMessage() error {
	mt, msg, err := r.conn.ReadMessage()
	if err != nil {
		return err
	}

	switch mt {
	case websocket.BinaryMessage:
		if err = r.persistTask(msg); err != nil {
			return err
		}
	}
	return nil
}

func (r *Roller) prove() error {
	task, err := r.stack.Pop()
	if err != nil {
		return err
	}

	var proofMsg *message.ProofMsg
	if task.Times > 2 {
		proofMsg = &message.ProofMsg{
			Status: message.StatusProofError,
			Error:  "prover has retried several times due to FFI panic",
			ID:     task.Task.ID,
			Proof:  &message.AggProof{},
		}
		return r.sendMessage(message.ProofMsgType, proofMsg)
	}

	err = r.stack.Push(task)
	if err != nil {
		return err
	}

	log.Info("start to prove block", "task-id", task.Task.ID)

	proof, err := r.prover.Prove(task.Task.Traces)
	if err != nil {
		proofMsg = &message.ProofMsg{
			Status: message.StatusProofError,
			Error:  err.Error(),
			ID:     task.Task.ID,
			Proof:  &message.AggProof{},
		}
		log.Error("prove block failed!", "task-id", task.Task.ID)
	} else {

		proofMsg = &message.ProofMsg{
			Status: message.StatusOk,
			ID:     task.Task.ID,
			Proof:  proof,
		}
		log.Info("prove block successfully!", "task-id", task.Task.ID)
	}
	_, err = r.stack.Pop()
	if err != nil {
		return err
	}

	return r.sendMessage(message.ProofMsgType, proofMsg)
}

// Close closes the websocket connection.
func (r *Roller) Close() {
	if atomic.LoadInt64(&r.isClosed) == 1 {
		return
	}
	atomic.StoreInt64(&r.isClosed, 1)

	close(r.stopChan)
	// Close coordinator's ws
	_ = r.conn.Close()
	// Close db
	if err := r.stack.Close(); err != nil {
		log.Error("failed to close bbolt db", "error", err)
	}
}

func (r *Roller) persistTask(byt []byte) error {
	var msg = &message.Msg{}
	err := json.Unmarshal(byt, msg)
	if err != nil {
		return err
	}
	if msg.Type != message.TaskMsgType {
		log.Error("message from coordinator illegal")
		return nil
	}
	var task = &message.TaskMsg{}
	if err := json.Unmarshal(msg.Payload, task); err != nil {
		return err
	}
	log.Info("Accept task from coordinator", "ID", task.ID)
	return r.stack.Push(&store.ProvingTask{
		Task:  task,
		Times: 0,
	})
}

func (r *Roller) loadOrCreateKey() (*ecdsa.PrivateKey, error) {
	keystoreFilePath := r.cfg.KeystorePath
	if _, err := os.Stat(r.cfg.KeystorePath); os.IsNotExist(err) {
		// If there is no keystore, make a new one.
		ks := keystore.NewKeyStore(r.cfg.KeystorePath, keystore.StandardScryptN, keystore.StandardScryptP)
		account, err := ks.NewAccount(r.cfg.KeystorePassword)
		if err != nil {
			return nil, fmt.Errorf("generate crypto account failed %v", err)
		}
		log.Info("create a new account", "address", account.Address.Hex())

		fis, err := ioutil.ReadDir(r.cfg.KeystorePath)
		if err != nil {
			return nil, err
		}
		keystoreFilePath = filepath.Join(r.cfg.KeystorePath, fis[0].Name())
	} else {
		return nil, err
	}

	keyjson, err := ioutil.ReadFile(keystoreFilePath)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(keyjson, r.cfg.KeystorePassword)
	if err != nil {
		return nil, err
	}
	return key.PrivateKey, nil
}

// MakeMsgByt Marshals Msg to bytes.
func MakeMsgByt(msgTyp message.MsgType, payloadVal interface{}) ([]byte, error) {
	payload, err := json.Marshal(payloadVal)
	if err != nil {
		return nil, err
	}
	msg := &message.Msg{
		Type:    msgTyp,
		Payload: payload,
	}
	return json.Marshal(msg)
}
