package core

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/keystore"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/message"
	"scroll-tech/common/version"

	"scroll-tech/coordinator/client"

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
	cfg       *config.Config
	client    *client.Client
	stack     *store.Stack
	prover    *prover.Prover
	traceChan chan *message.TaskMsg
	sub       ethereum.Subscription

	isClosed int64
	stopChan chan struct{}

	priv *ecdsa.PrivateKey
}

// NewRoller new a Roller object.
func NewRoller(cfg *config.Config) (*Roller, error) {
	// load or create wallet
	priv, err := loadOrCreateKey(cfg.KeystorePath, cfg.KeystorePassword)
	if err != nil {
		return nil, err
	}

	// Get stack db handler
	stackDb, err := store.NewStack(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	// Create prover instance
	log.Info("init prover")
	prover, err := prover.NewProver(cfg.Prover)
	if err != nil {
		return nil, err
	}
	log.Info("init prover successfully!")

	rClient, err := client.Dial(cfg.CoordinatorURL)

	if err != nil {
		return nil, err
	}

	return &Roller{
		cfg:       cfg,
		client:    rClient,
		stack:     stackDb,
		prover:    prover,
		sub:       nil,
		traceChan: make(chan *message.TaskMsg, 10),
		stopChan:  make(chan struct{}),
		priv:      priv,
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
	authMsg := &message.AuthMessage{
		Identity: &message.Identity{
			Name:      r.cfg.RollerName,
			Timestamp: time.Now().UnixMilli(),
			PublicKey: common.Bytes2Hex(crypto.CompressPubkey(&r.priv.PublicKey)),
			Version:   Version,
			Ticket:    nil,
		},
	}
	// No need to sing authMsg here, manager can extract roller's PublicKey from corresponding field in Identity
	// if err := authMsg.Sign(r.priv); err != nil {
	// 	return fmt.Errorf("sign auth message failed %v", err)
	// }

	ticket, err := r.client.RequestTicket(context.Background(), authMsg)
	if err != nil {
		return fmt.Errorf("request ticket failed %v", err)
	} else {
		authMsg.Identity.Ticket = &ticket
	}

	// Sign auth message
	if err := authMsg.Sign(r.priv); err != nil {
		return fmt.Errorf("sign auth message failed %v", err)
	}

	sub, err := r.client.RegisterAndSubscribe(context.Background(), r.traceChan, authMsg)
	r.sub = sub
	return err
}

// HandleCoordinator accepts block-traces from coordinator through the Websocket and store it into Stack.
func (r *Roller) HandleCoordinator() {
	for {
		select {
		case <-r.stopChan:
			return
		case trace := <-r.traceChan:
			log.Info("Accept BlockTrace from Scroll", "ID", trace.ID)
			err := r.stack.Push(&store.ProvingTask{Task: trace, Times: 0})
			if err != nil {
				panic(fmt.Sprintf("could not push trace(%s) into stack: %v", trace.ID, err))
			}
		case err := <-r.sub.Err():
			r.sub.Unsubscribe()
			log.Error("Subscribe trace with scroll failed", "error", err)
			r.mustRetryCoordinator()
		}
	}
}

func (r *Roller) mustRetryCoordinator() {
	for {
		log.Info("retry to connect to coordinator...")
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

		_, err = r.signAndSubmitProof(proofMsg)
		return err
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

	ok, err := r.signAndSubmitProof(proofMsg)
	if !ok {
		log.Error("Submit proof to scroll failed! auzhZkProofID: ", proofMsg.ID)
	}
	return err
}

func (r *Roller) signAndSubmitProof(msg *message.ProofMsg) (bool, error) {
	authZkProof := &message.AuthZkProof{ProofMsg: msg}
	if err := authZkProof.Sign(r.priv); err != nil {
		return false, err
	}

	return r.client.SubmitProof(context.Background(), authZkProof)
}

// Close closes the websocket connection.
func (r *Roller) Close() {
	if atomic.LoadInt64(&r.isClosed) == 1 {
		return
	}
	atomic.StoreInt64(&r.isClosed, 1)

	close(r.stopChan)
	// Close scroll's ws
	r.sub.Unsubscribe()
	// Close db
	if err := r.stack.Close(); err != nil {
		log.Error("failed to close bbolt db", "error", err)
	}
}

func loadOrCreateKey(keystoreDir string, keystorePassword string) (*ecdsa.PrivateKey, error) {
	if _, err := os.Stat(keystoreDir); os.IsNotExist(err) {
		// If there is no keystore, make a new one.
		ks := keystore.NewKeyStore(keystoreDir, keystore.StandardScryptN, keystore.StandardScryptP)
		account, err := ks.NewAccount(keystorePassword)
		if err != nil {
			return nil, fmt.Errorf("generate crypto account failed %v", err)
		}
		log.Info("create a new account", "address", account.Address.Hex())
	} else if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(keystoreDir)
	if err != nil {
		return nil, err
	}
	keystorePath := filepath.Join(keystoreDir, entries[0].Name())
	keyjson, err := ioutil.ReadFile(keystorePath)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(keyjson, keystorePassword)
	if err != nil {
		return nil, err
	}
	return key.PrivateKey, nil
}
