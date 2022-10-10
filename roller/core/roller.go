package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/go-roller/client"
	"scroll-tech/go-roller/config"
	"scroll-tech/go-roller/core/prover"
	"scroll-tech/go-roller/message"
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
	cfg       *config.Config
	client    *client.Client
	stack     *store.Stack
	prover    *prover.Prover
	traceChan chan *message.BlockTraces
	sub       ethereum.Subscription

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
	prover, err := prover.NewProver(cfg.Prover)
	if err != nil {
		return nil, err
	}
	log.Info("init prover successfully!")

	rClient, err := client.Dial(cfg.ScrollURL)
	if err != nil {
		return nil, err
	}

	return &Roller{
		cfg:       cfg,
		client:    rClient,
		stack:     stackDb,
		prover:    prover,
		sub:       nil,
		traceChan: make(chan *message.BlockTraces, 2),
		stopChan:  make(chan struct{}),
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
	priv, err := crypto.HexToECDSA(r.cfg.SecretKey)
	if err != nil {
		return fmt.Errorf("generate private-key failed %v", err)
	}
	authMsg := &message.AuthMessage{
		Identity: &message.Identity{
			Name:      r.cfg.RollerName,
			Timestamp: time.Now().UnixMilli(),
		},
	}

	// Sign auth message
	if err = authMsg.Sign(priv); err != nil {
		return fmt.Errorf("sign auth message failed %v", err)
	}

	sub, err := r.client.SubscribeRegister(context.Background(), r.traceChan, authMsg)
	r.sub = sub
	return err
}

// HandleScroll accepts block-traces from Scroll through the Websocket and store it into Stack.
func (r *Roller) HandleScroll() {
	for {
		select {
		case <-r.stopChan:
			return
		case trace := <-r.traceChan:
			log.Info("Accept BlockTrace from Scroll", "ID")
			err := r.stack.Push(trace)
			if err != nil {
				panic(fmt.Sprintf("could not push trace(%d) into stack: %v", trace.ID, err))
			}
		case err := <-r.sub.Err():
			r.sub.Unsubscribe()
			log.Error("Subscribe trace with scroll failed", "error", err)
			r.mustRetryScroll()
		}
	}
}

func (r *Roller) mustRetryScroll() {
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
	traces, err := r.stack.Pop()
	if err != nil {
		return err
	}
	log.Info("start to prove block", "block-Number", traces.ID)

	var proofMsg *message.ProofMsg
	proof, err := r.prover.Prove(traces.Traces)
	if err != nil {
		proofMsg = &message.ProofMsg{
			Status: message.StatusProofError,
			Error:  err.Error(),
			ID:     traces.ID,
			Proof:  &message.AggProof{},
		}
		log.Error("prove block failed!", "block-id", traces.ID)
	} else {
		proofMsg = &message.ProofMsg{
			Status: message.StatusOk,
			ID:     traces.ID,
			Proof:  proof,
		}
		log.Info("prove block successfully!", "block-id", traces.ID)
	}

	priv, err := crypto.HexToECDSA(r.cfg.SecretKey)
	if err != nil {
		return fmt.Errorf("generate private-key failed %v", err)
	}

	authZkProof := &message.AuthZkProof{ProofMsg: proofMsg}
	if err = authZkProof.Sign(priv); err != nil {
		return err
	}

	ok, err := r.client.SubmitProof(context.Background(), authZkProof)
	if !ok {
		log.Error("Submit proof to scroll failed! auzhZkProofID: ", authZkProof.ID)
	}
	return err
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
