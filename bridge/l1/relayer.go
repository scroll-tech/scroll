package l1

import (
	"context"
	"errors"
	"math/big"
	"time"

	// not sure if this will make problems when relay with l1geth

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database/orm"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
)

// Layer1Relayer is responsible for
//  1. fetch pending L1Message from db
//  2. relay pending message to layer 2 node
//
// Actions are triggered by new head from layer 1 geth node.
// @todo It's better to be triggered by watcher.
type Layer1Relayer struct {
	ctx    context.Context
	client *ethclient.Client
	sender *sender.Sender

	db  orm.L1MessageOrm
	cfg *config.RelayerConfig

	// channel used to communicate with transaction sender
	confirmationCh <-chan *sender.Confirmation
	l2MessengerABI *abi.ABI

	stopCh chan struct{}
}

// NewLayer1Relayer will return a new instance of Layer1RelayerClient
func NewLayer1Relayer(ctx context.Context, ethClient *ethclient.Client, l1ConfirmNum int64, db orm.L1MessageOrm, cfg *config.RelayerConfig) (*Layer1Relayer, error) {
	l2MessengerABI, err := bridge_abi.L2MessengerMetaData.GetAbi()
	if err != nil {
		log.Warn("new L2MessengerABI failed", "err", err)
		return nil, err
	}

	sender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.MessageSenderPrivateKeys)
	if err != nil {
		addr := crypto.PubkeyToAddress(cfg.MessageSenderPrivateKeys[0].PublicKey)
		log.Error("new sender failed", "address", addr.String(), "err", err)
		return nil, err
	}

	return &Layer1Relayer{
		ctx:            ctx,
		client:         ethClient,
		sender:         sender,
		db:             db,
		l2MessengerABI: l2MessengerABI,
		cfg:            cfg,
		stopCh:         make(chan struct{}),
		confirmationCh: sender.ConfirmChan(),
	}, nil
}

// ProcessSavedEvents relays saved un-processed cross-domain transactions to desired blockchain
func (r *Layer1Relayer) ProcessSavedEvents() {
	// msgs are sorted by nonce in increasing order
	msgs, err := r.db.GetL1MessagesByStatus(orm.MsgPending)
	if err != nil {
		log.Error("Failed to fetch unprocessed L1 messages", "err", err)
		return
	}
	for _, msg := range msgs {
		if err = r.processSavedEvent(msg); err != nil {
			if !errors.Is(err, sender.ErrNoAvailableAccount) {
				log.Error("failed to process event", "err", err)
			}
			return
		}
	}
}

func (r *Layer1Relayer) processSavedEvent(msg *orm.L1Message) error {
	// @todo add support to relay multiple messages
	from := common.HexToAddress(msg.Sender)
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
	data, err := r.l2MessengerABI.Pack("relayMessage", from, target, value, fee, deadline, msgNonce, calldata)
	if err != nil {
		log.Error("Failed to pack relayMessage", "msg.nonce", msg.Nonce, "msg.height", msg.Height, "err", err)
		// TODO: need to skip this message by changing its status to MsgError
		return err
	}

	hash, err := r.sender.SendTransaction(msg.MsgHash, &r.cfg.MessengerContractAddress, big.NewInt(0), data)
	if err != nil {
		return err
	}
	log.Info("relayMessage to layer2", "msg hash", msg.MsgHash, "tx hash", hash)

	err = r.db.UpdateLayer1StatusAndLayer2Hash(r.ctx, msg.MsgHash, orm.MsgSubmitted, hash.String())
	if err != nil {
		log.Error("UpdateLayer1StatusAndLayer2Hash failed", "msg.msgHash", msg.MsgHash, "msg.height", msg.Height, "err", err)
	}
	return err
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
			case cfm := <-r.confirmationCh:
				if !cfm.IsSuccessful {
					log.Warn("transaction confirmed but failed in layer2", "confirmation", cfm)
				} else {
					// @todo handle db error
					err := r.db.UpdateLayer1StatusAndLayer2Hash(r.ctx, cfm.ID, orm.MsgConfirmed, cfm.TxHash.String())
					if err != nil {
						log.Warn("UpdateLayer1StatusAndLayer2Hash failed", "err", err)
					}
					log.Info("transaction confirmed in layer2", "confirmation", cfm)
				}
			case <-r.stopCh:
				return
			}
		}
	}()
}

// Stop the relayer module, for a graceful shutdown.
func (r *Layer1Relayer) Stop() {
	close(r.stopCh)
}
