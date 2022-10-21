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

	message "scroll-tech/common/message"

	"scroll-tech/go-roller/config"
	"scroll-tech/go-roller/core/prover"
	"scroll-tech/go-roller/store"
)

var (
	writeWait = time.Second + readWait
	// consider ping message
	readWait = time.Minute * 30
	// retry scroll
	retryWait = time.Second * 10
	// net normal close
	errNormalClose = errors.New("use of closed network connection")
)

// Roller contains websocket conn to Scroll, Stack, unix-socket to ipc-prover.
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

	conn, _, err := websocket.DefaultDialer.Dial(cfg.ScrollURL, nil)
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
	log.Info("start to register to scroll")
	if err := r.Register(); err != nil {
		log.Crit("register to scroll failed", "error", err)
	}
	log.Info("register to scroll successfully!")
	go func() {
		r.HandleScroll()
		r.Close()
	}()

	return r.ProveLoop()
}

// Register registers Roller to the Scroll through Websocket.
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
		},
		Signature: "",
	}

	// Sign auth message
	if err = authMsg.Sign(priv); err != nil {
		return fmt.Errorf("sign auth message failed %v", err)
	}

	return r.sendMessage(message.Register, authMsg)
}

// HandleScroll accepts block-traces from Scroll through the Websocket and store it into Stack.
func (r *Roller) HandleScroll() {
	for {
		select {
		case <-r.stopChan:
			return
		default:
			_ = r.conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = r.conn.SetReadDeadline(time.Now().Add(readWait))
			if err := r.handMessage(); err != nil && !strings.Contains(err.Error(), errNormalClose.Error()) {
				log.Error("handle scroll failed", "error", err)
				r.mustRetryScroll()
				continue
			}
		}
	}
}

func (r *Roller) mustRetryScroll() {
	for {
		log.Info("retry to connect to scroll...")
		conn, _, err := websocket.DefaultDialer.Dial(r.cfg.ScrollURL, nil)
		if err != nil {
			log.Error("failed to connect scroll: ", "error", err)
			time.Sleep(retryWait)
		} else {
			r.conn = conn
			log.Info("re-connect to scroll successfully!")
			break
		}
	}
	for {
		log.Info("retry to register to scroll...")
		err := r.Register()
		if err != nil {
			log.Error("register to scroll failed", "error", err)
			time.Sleep(retryWait)
		} else {
			log.Info("re-register to scroll successfully!")
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
		if err = r.persistTrace(msg); err != nil {
			return err
		}
	}
	return nil
}

func (r *Roller) prove() error {
	traces, err := r.stack.Pop()
	if err != nil {
		return err
	}

	var proofMsg *message.ProofMsg
	if traces.Times > 2 {
		proofMsg = &message.ProofMsg{
			Status: message.StatusProofError,
			Error:  "prover has retried several times due to FFI panic",
			ID:     traces.Traces.ID,
			Proof:  &message.AggProof{},
		}
		return r.sendMessage(message.Proof, proofMsg)
	}

	err = r.stack.Push(traces)
	if err != nil {
		return err
	}

	log.Info("start to prove block", "block-id", traces.Traces.ID)

	proof, err := r.prover.Prove(traces.Traces.Traces)
	if err != nil {
		proofMsg = &message.ProofMsg{
			Status: message.StatusProofError,
			Error:  err.Error(),
			ID:     traces.Traces.ID,
			Proof:  &message.AggProof{},
		}
		log.Error("prove block failed!", "block-id", traces.Traces.ID)
	} else {

		proofMsg = &message.ProofMsg{
			Status: message.StatusOk,
			ID:     traces.Traces.ID,
			Proof:  proof,
		}
		log.Info("prove block successfully!", "block-id", traces.Traces.ID)
	}
	_, err = r.stack.Pop()
	if err != nil {
		return err
	}

	return r.sendMessage(message.Proof, proofMsg)
}

// Close closes the websocket connection.
func (r *Roller) Close() {
	if atomic.LoadInt64(&r.isClosed) == 1 {
		return
	}
	atomic.StoreInt64(&r.isClosed, 1)

	close(r.stopChan)
	// Close scroll's ws
	_ = r.conn.Close()
	// Close db
	if err := r.stack.Close(); err != nil {
		log.Error("failed to close bbolt db", "error", err)
	}
}

func (r *Roller) persistTrace(byt []byte) error {
	var msg = &message.Msg{}
	err := json.Unmarshal(byt, msg)
	if err != nil {
		return err
	}
	if msg.Type != message.BlockTrace {
		log.Error("message from Scroll illegal")
		return nil
	}
	var traces = &message.BlockTraces{}
	if err := json.Unmarshal(msg.Payload, traces); err != nil {
		return err
	}
	log.Info("Accept BlockTrace from Scroll", "ID", traces.ID)
	return r.stack.Push(&store.ProvingTraces{
		Traces: traces,
		Times:  0,
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
