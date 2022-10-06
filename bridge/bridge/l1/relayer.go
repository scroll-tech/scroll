package l1

import (
	"context"
	"math/big"
	"time"

	// not sure if this will make problems when relay with l1geth

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	bridge_abi "scroll-tech/bridge/bridge/abi"
	"scroll-tech/bridge/bridge/sender"
	"scroll-tech/bridge/config"
	"scroll-tech/store/orm"
)

var (
	// BufferSize for the Transaction Confirmation channel
	confirmationChanSize = 100
)

// Layer1Relayer is responsible for
//  1. fetch pending Layer1Message from db
//  2. relay pending message to layer 2 node
//
// Actions are triggered by new head from layer 1 geth node.
// @todo It's better to be triggered by watcher.
type Layer1Relayer struct {
	ctx    context.Context
	client *ethclient.Client
	sender *sender.Sender

	db  orm.Layer1MessageOrm
	cfg *config.RelayerConfig

	l2MessengerABI *abi.ABI

	// a list of processing message, from tx.nonce to message nonce
	processingMessage map[uint64]uint64

	// channel used to communicate with transaction sender
	confirmationCh chan *sender.Confirmation
	stop           chan bool
}

// NewLayer1Relayer will return a new instance of Layer1RelayerClient
func NewLayer1Relayer(ctx context.Context, ethClient *ethclient.Client, db orm.Layer1MessageOrm, cfg *config.RelayerConfig) (*Layer1Relayer, error) {

	confirmationCh := make(chan *sender.Confirmation, confirmationChanSize)

	l2MessengerABI, err := bridge_abi.L2MessengerMetaData.GetAbi()
	if err != nil {
		log.Warn("new L2MessengerABI failed", "err", err)
		return nil, err
	}

	prv, err := crypto.HexToECDSA(cfg.PrivateKey)
	if err != nil {
		log.Error("Failed to import private key from config file")
		return nil, err
	}

	sender, err := sender.NewSender(ctx, confirmationCh, *cfg.SenderConfig, prv)
	if err != nil {
		log.Error("new sender failed", "err", err)
		return nil, err
	}

	return &Layer1Relayer{
		ctx:               ctx,
		client:            ethClient,
		sender:            sender,
		db:                db,
		l2MessengerABI:    l2MessengerABI,
		cfg:               cfg,
		processingMessage: map[uint64]uint64{},
		confirmationCh:    confirmationCh,
		stop:              make(chan bool),
	}, nil
}

// ProcessSavedEvents relays saved un-processed cross-domain transactions to desired blockchain
func (r *Layer1Relayer) ProcessSavedEvents() {
	// msgs are sorted by nonce in increasing order
	msgs, err := r.db.GetL1UnprocessedMessages()
	if err != nil {
		log.Error("Failed to fetch unprocessed L1 messages", "err", err)
		return
	}
	if len(msgs) == 0 {
		return
	}
	msg := msgs[0]
	// @todo add support to relay multiple messages
	sender := common.HexToAddress(msg.Sender)
	target := common.HexToAddress(msg.Target)
	value, ok := big.NewInt(0).SetString(msg.Value, 10)
	if !ok {
		// @todo maybe panic?
		log.Error("Failed to parse message value", "msg.nonce", msg.Nonce, "msg.height", msg.Height)
		// TODO: need to skip this message by changing its status to MsgError
	}
	fee, _ := big.NewInt(0).SetString(msg.Fee, 10)
	deadline := big.NewInt(int64(msg.Deadline))
	msgNonce := big.NewInt(int64(msg.Nonce))
	calldata := common.Hex2Bytes(msg.Calldata)
	data, err := r.l2MessengerABI.Pack("relayMessage", sender, target, value, fee, deadline, msgNonce, calldata)
	if err != nil {
		log.Error("Failed to pack relayMessage", "msg.nonce", msg.Nonce, "msg.height", msg.Height, "err", err)
		// TODO: need to skip this message by changing its status to MsgError
		return
	}

	nonce, hash, err := r.sender.SendTransaction(&r.cfg.MessengerContractAddress, big.NewInt(0), data)
	if err != nil {
		log.Error("Failed to send relayMessage tx to L2", "msg.nonce", msg.Nonce, "msg.height", msg.Height, "err", err)
		return
	}
	log.Info("relayMessage to layer2", "tx.nonce", nonce, "hash", hash)

	// save status in db
	// @todo handle db error
	err = r.db.UpdateLayer1StatusAndLayer2Hash(r.ctx, msg.Nonce, hash.String(), orm.MsgSubmitted)
	if err != nil {
		log.Error("UpdateLayer1StatusAndLayer2Hash failed", "msg.nonce", msg.Nonce, "msg.height", msg.Height, "err", err)
	}
	r.processingMessage[nonce] = msg.Nonce
}

// Start the relayer process
func (r *Layer1Relayer) Start() {
	go func() {
		// trigger by timer
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// number, err := r.client.BlockNumber(r.ctx)
				// log.Info("receive header", "height", number)
				r.ProcessSavedEvents()
			case confirmation := <-r.confirmationCh:
				msgNonce := r.processingMessage[confirmation.Nonce]
				// @todo handle db error
				err := r.db.UpdateLayer1StatusAndLayer2Hash(r.ctx, msgNonce, confirmation.Hash.String(), orm.MsgConfirmed)
				if err != nil {
					log.Warn("UpdateLayer1StatusAndLayer2Hash failed", "err", err)
				}
				delete(r.processingMessage, confirmation.Nonce)
				log.Info("transaction confirmed in layer2", "confirmation", confirmation)
			case <-r.stop:
				return
			}
		}
	}()
}

// Stop the relayer module, for a graceful shutdown.
func (r *Layer1Relayer) Stop() {
	r.stop <- true
}
