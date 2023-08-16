package relayer

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	// not sure if this will make problems when relay with l1geth
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	bridgeAbi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/controller/sender"
	"scroll-tech/bridge/internal/orm"
)

// Layer1Relayer is responsible for
//  1. fetch pending L1Message from db
//  2. relay pending message to layer 2 node
//
// Actions are triggered by new head from layer 1 geth node.
// @todo It's better to be triggered by watcher.
type Layer1Relayer struct {
	ctx context.Context

	cfg *config.RelayerConfig

	// channel used to communicate with transaction sender
	messageSender  *sender.Sender
	l2MessengerABI *abi.ABI

	gasOracleSender *sender.Sender
	l1GasOracleABI  *abi.ABI

	minGasLimitForMessageRelay uint64

	lastGasPrice uint64
	minGasPrice  uint64
	gasPriceDiff uint64

	l1MessageOrm *orm.L1Message
	l1BlockOrm   *orm.L1Block

	bridgeL1RelayedMsgsTotal               prometheus.Counter
	bridgeL1RelayedMsgsFailureTotal        prometheus.Counter
	bridgeL1RelayerGasPriceOraclerRunTotal prometheus.Counter
	bridgeL1RelayerLastGasPrice            prometheus.Gauge
	bridgeL1MsgsRelayedConfirmedTotal      prometheus.Counter
	bridgeL1GasOraclerConfirmedTotal       prometheus.Counter
}

// NewLayer1Relayer will return a new instance of Layer1RelayerClient
func NewLayer1Relayer(ctx context.Context, db *gorm.DB, cfg *config.RelayerConfig, reg prometheus.Registerer) (*Layer1Relayer, error) {
	messageSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.MessageSenderPrivateKey, "l1_relayer", "message_sender", reg)
	if err != nil {
		addr := crypto.PubkeyToAddress(cfg.MessageSenderPrivateKey.PublicKey)
		return nil, fmt.Errorf("new message sender failed for address %s, err: %v", addr.Hex(), err)
	}

	gasOracleSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.GasOracleSenderPrivateKey, "l1_relayer", "gas_oracle_sender", reg)
	if err != nil {
		addr := crypto.PubkeyToAddress(cfg.GasOracleSenderPrivateKey.PublicKey)
		return nil, fmt.Errorf("new gas oracle sender failed for address %s, err: %v", addr.Hex(), err)
	}

	var minGasPrice uint64
	var gasPriceDiff uint64
	if cfg.GasOracleConfig != nil {
		minGasPrice = cfg.GasOracleConfig.MinGasPrice
		gasPriceDiff = cfg.GasOracleConfig.GasPriceDiff
	} else {
		minGasPrice = 0
		gasPriceDiff = defaultGasPriceDiff
	}

	minGasLimitForMessageRelay := uint64(defaultL1MessageRelayMinGasLimit)
	if cfg.MessageRelayMinGasLimit != 0 {
		minGasLimitForMessageRelay = cfg.MessageRelayMinGasLimit
	}

	l1Relayer := &Layer1Relayer{
		cfg:          cfg,
		ctx:          ctx,
		l1MessageOrm: orm.NewL1Message(db),
		l1BlockOrm:   orm.NewL1Block(db),

		messageSender:  messageSender,
		l2MessengerABI: bridgeAbi.L2ScrollMessengerABI,

		gasOracleSender: gasOracleSender,
		l1GasOracleABI:  bridgeAbi.L1GasPriceOracleABI,

		minGasLimitForMessageRelay: minGasLimitForMessageRelay,

		minGasPrice:  minGasPrice,
		gasPriceDiff: gasPriceDiff,

		bridgeL1RelayedMsgsTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_layer1_msg_relayed_total",
			Help: "The total number of the l1 relayed message.",
		}),
		bridgeL1RelayedMsgsFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_layer1_msg_relayed_failure_total",
			Help: "The total number of the l1 relayed failure message.",
		}),
		bridgeL1MsgsRelayedConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_layer1_relayed_confirmed_total",
			Help: "The total number of layer1 relayed confirmed",
		}),
		bridgeL1RelayerGasPriceOraclerRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_layer1_gas_price_oracler_total",
			Help: "The total number of layer1 gas price oracler run total",
		}),
		bridgeL1RelayerLastGasPrice: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "bridge_layer1_gas_price_latest_gas_price",
			Help: "The latest gas price of bridge relayer l1",
		}),
		bridgeL1GasOraclerConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "bridge_layer1_relayed_confirmed_total",
			Help: "The total number of layer1 relayed confirmed",
		}),
	}

	go l1Relayer.handleConfirmLoop(ctx)
	return l1Relayer, nil
}

// ProcessSavedEvents relays saved un-processed cross-domain transactions to desired blockchain
func (r *Layer1Relayer) ProcessSavedEvents() {
	// msgs are sorted by nonce in increasing order
	msgs, err := r.l1MessageOrm.GetL1MessagesByStatus(types.MsgPending, 100)
	if err != nil {
		log.Error("Failed to fetch unprocessed L1 messages", "err", err)
		return
	}

	if len(msgs) > 0 {
		log.Info("Processing L1 messages", "count", len(msgs))
	}

	for _, msg := range msgs {
		tmpMsg := msg
		r.bridgeL1RelayedMsgsTotal.Inc()
		if err = r.processSavedEvent(&tmpMsg); err != nil {
			r.bridgeL1RelayedMsgsFailureTotal.Inc()
			if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
				log.Error("failed to process event", "msg.msgHash", msg.MsgHash, "err", err)
			}
			return
		}
	}
}

func (r *Layer1Relayer) processSavedEvent(msg *orm.L1Message) error {
	calldata := common.Hex2Bytes(msg.Calldata)
	hash, err := r.messageSender.SendTransaction(msg.MsgHash, &r.cfg.MessengerContractAddress, big.NewInt(0), calldata, r.minGasLimitForMessageRelay)
	if err != nil && errors.Is(err, ErrExecutionRevertedMessageExpired) {
		return r.l1MessageOrm.UpdateLayer1Status(r.ctx, msg.MsgHash, types.MsgExpired)
	}

	if err != nil && errors.Is(err, ErrExecutionRevertedAlreadySuccessExecuted) {
		return r.l1MessageOrm.UpdateLayer1Status(r.ctx, msg.MsgHash, types.MsgConfirmed)
	}
	if err != nil {
		return err
	}
	log.Info("relayMessage to layer2", "msg hash", msg.MsgHash, "tx hash", hash)

	err = r.l1MessageOrm.UpdateLayer1StatusAndLayer2Hash(r.ctx, msg.MsgHash, types.MsgSubmitted, hash.String())
	if err != nil {
		log.Error("UpdateLayer1StatusAndLayer2Hash failed", "msg.msgHash", msg.MsgHash, "msg.height", msg.Height, "err", err)
	}
	return err
}

// ProcessGasPriceOracle imports gas price to layer2
func (r *Layer1Relayer) ProcessGasPriceOracle() {
	r.bridgeL1RelayerGasPriceOraclerRunTotal.Inc()
	latestBlockHeight, err := r.l1BlockOrm.GetLatestL1BlockHeight(r.ctx)
	if err != nil {
		log.Warn("Failed to fetch latest L1 block height from db", "err", err)
		return
	}

	blocks, err := r.l1BlockOrm.GetL1Blocks(r.ctx, map[string]interface{}{
		"number": latestBlockHeight,
	})
	if err != nil {
		log.Error("Failed to GetL1Blocks from db", "height", latestBlockHeight, "err", err)
		return
	}
	if len(blocks) != 1 {
		log.Error("Block not exist", "height", latestBlockHeight)
		return
	}
	block := blocks[0]

	if types.GasOracleStatus(block.GasOracleStatus) == types.GasOraclePending {
		expectedDelta := r.lastGasPrice * r.gasPriceDiff / gasPriceDiffPrecision
		// last is undefine or (block.BaseFee >= minGasPrice && exceed diff)
		if r.lastGasPrice == 0 || (block.BaseFee >= r.minGasPrice && (block.BaseFee >= r.lastGasPrice+expectedDelta || block.BaseFee <= r.lastGasPrice-expectedDelta)) {
			baseFee := big.NewInt(int64(block.BaseFee))
			data, err := r.l1GasOracleABI.Pack("setL1BaseFee", baseFee)
			if err != nil {
				log.Error("Failed to pack setL1BaseFee", "block.Hash", block.Hash, "block.Height", block.Number, "block.BaseFee", block.BaseFee, "err", err)
				return
			}

			hash, err := r.gasOracleSender.SendTransaction(block.Hash, &r.cfg.GasPriceOracleContractAddress, big.NewInt(0), data, 0)
			if err != nil {
				if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
					log.Error("Failed to send setL1BaseFee tx to layer2 ", "block.Hash", block.Hash, "block.Height", block.Number, "err", err)
				}
				return
			}

			err = r.l1BlockOrm.UpdateL1GasOracleStatusAndOracleTxHash(r.ctx, block.Hash, types.GasOracleImporting, hash.String())
			if err != nil {
				log.Error("UpdateGasOracleStatusAndOracleTxHash failed", "block.Hash", block.Hash, "block.Height", block.Number, "err", err)
				return
			}
			r.lastGasPrice = block.BaseFee
			r.bridgeL1RelayerLastGasPrice.Set(float64(r.lastGasPrice))
			log.Info("Update l1 base fee", "txHash", hash.String(), "baseFee", baseFee)
		}
	}
}

func (r *Layer1Relayer) handleConfirmLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cfm := <-r.messageSender.ConfirmChan():
			r.bridgeL1MsgsRelayedConfirmedTotal.Inc()
			if !cfm.IsSuccessful {
				err := r.l1MessageOrm.UpdateLayer1StatusAndLayer2Hash(r.ctx, cfm.ID, types.MsgRelayFailed, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateLayer1StatusAndLayer2Hash failed", "err", err)
				}
				log.Warn("transaction confirmed but failed in layer2", "confirmation", cfm)
			} else {
				// @todo handle db error
				err := r.l1MessageOrm.UpdateLayer1StatusAndLayer2Hash(r.ctx, cfm.ID, types.MsgConfirmed, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateLayer1StatusAndLayer2Hash failed", "err", err)
				}
				log.Info("transaction confirmed in layer2", "confirmation", cfm)
			}
		case cfm := <-r.gasOracleSender.ConfirmChan():
			r.bridgeL1MsgsRelayedConfirmedTotal.Inc()
			if !cfm.IsSuccessful {
				// @discuss: maybe make it pending again?
				err := r.l1BlockOrm.UpdateL1GasOracleStatusAndOracleTxHash(r.ctx, cfm.ID, types.GasOracleFailed, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateL1GasOracleStatusAndOracleTxHash failed", "err", err)
				}
				log.Warn("transaction confirmed but failed in layer2", "confirmation", cfm)
			} else {
				// @todo handle db error
				err := r.l1BlockOrm.UpdateL1GasOracleStatusAndOracleTxHash(r.ctx, cfm.ID, types.GasOracleImported, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateGasOracleStatusAndOracleTxHash failed", "err", err)
				}
				log.Info("transaction confirmed in layer2", "confirmation", cfm)
			}
		}
	}
}
