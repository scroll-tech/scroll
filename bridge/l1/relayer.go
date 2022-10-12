package l1

import (
	"context"
	"math/big"
	"sync/atomic"
	"time"

	// not sure if this will make problems when relay with l1geth

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/store/orm"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
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

	//checked block height
	l1ConfirmNum          int64
	checkedPendingTxBlock uint64

	// channel used to communicate with transaction sender
	confirmationCh <-chan *sender.Confirmation
	l2MessengerABI *abi.ABI

	stopCh chan struct{}
}

// NewLayer1Relayer will return a new instance of Layer1RelayerClient
func NewLayer1Relayer(ctx context.Context, ethClient *ethclient.Client, l1ConfirmNum int64, db orm.Layer1MessageOrm, cfg *config.RelayerConfig) (*Layer1Relayer, error) {
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

	sender, err := sender.NewSender(ctx, cfg.SenderConfig, prv)
	if err != nil {
		log.Error("new sender failed", "err", err)
		return nil, err
	}

	checkedPendingTxBlock, err := ethClient.BlockNumber(ctx)
	if err != nil {
		log.Error("Failed to get block number in l1/relayer", "err", err)
		return nil, err
	}

	return &Layer1Relayer{
		ctx:                   ctx,
		client:                ethClient,
		sender:                sender,
		db:                    db,
		l2MessengerABI:        l2MessengerABI,
		cfg:                   cfg,
		l1ConfirmNum:          l1ConfirmNum,
		checkedPendingTxBlock: checkedPendingTxBlock,
		stopCh:                make(chan struct{}),
		confirmationCh:        sender.ConfirmChan(),
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

	hash, err := r.sender.SendTransaction(msg.Layer1Hash, &r.cfg.MessengerContractAddress, big.NewInt(0), data)
	if err != nil {
		log.Error("Failed to send relayMessage tx to L2", "msg.nonce", msg.Nonce, "msg.height", msg.Height, "err", err)
		return
	}
	log.Info("relayMessage to layer2", "layer1 hash", msg.Layer1Hash, "tx hash", hash)

	err = r.db.UpdateLayer1StatusAndLayer2Hash(r.ctx, msg.Layer1Hash, hash.String(), orm.MsgSubmitted)
	if err != nil {
		log.Error("UpdateLayer1StatusAndLayer2Hash failed", "msg.layer1hash", msg.Layer1Hash, "msg.height", msg.Height, "err", err)
	}
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
				err := r.checkPendingTransaction()
				if err != nil {
					log.Warn("Can not check the pending Transaction from l1 relayer", "err", err)
				}
				r.ProcessSavedEvents()
			case cfm := <-r.confirmationCh: // @todo handle db error
				err := r.db.UpdateLayer1StatusAndLayer2Hash(r.ctx, cfm.ID, cfm.TxHash.String(), orm.MsgConfirmed)
				if err != nil {
					log.Warn("UpdateLayer1StatusAndLayer2Hash failed", "err", err)
				}
				log.Info("transaction confirmed in layer2", "confirmation", cfm)
			case <-r.stopCh:
				return
			}
		}
	}()
}

func (r *Layer1Relayer) checkPendingTransaction() error {
	uBlockNum, err := r.client.BlockNumber(r.ctx)
	if err != nil {
		return err
	}
	blockNum := int64(uBlockNum) - r.l1ConfirmNum
	if blockNum < 0 {
		blockNum = 0
	}
	for i := int64(r.checkedPendingTxBlock + 1); i <= blockNum; i++ {
		block, err := r.client.BlockByNumber(r.ctx, big.NewInt(i))
		if err != nil {
			return err
		}
		r.sender.CheckPendingTransaction(block)
		atomic.AddUint64(&r.checkedPendingTxBlock, 1)
	}
	return nil
}

// Stop the relayer module, for a graceful shutdown.
func (r *Layer1Relayer) Stop() {
	close(r.stopCh)
}
