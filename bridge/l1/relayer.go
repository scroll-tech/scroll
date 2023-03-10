package l1

import (
	"context"
	"time"

	// not sure if this will make problems when relay with l1geth

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"
	"scroll-tech/common/utils"

	"scroll-tech/database"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
)

const (
	gasPriceDiffPrecision = 1000000

	defaultGasPriceDiff = 50000 // 5%
)

// Layer1Relayer is responsible for
//  1. fetch pending L1Message from db
//  2. relay pending message to layer 2 node
//
// Actions are triggered by new head from layer 1 geth node.
// @todo It's better to be triggered by watcher.
type Layer1Relayer struct {
	ctx context.Context

	db  database.OrmFactory
	cfg *config.RelayerConfig

	// channel used to communicate with transaction sender
	messageSender  *sender.Sender
	messageCh      <-chan *sender.Confirmation
	l2MessengerABI *abi.ABI

	gasOracleSender *sender.Sender
	gasOracleCh     <-chan *sender.Confirmation
	l1GasOracleABI  *abi.ABI

	lastGasPrice uint64
	minGasPrice  uint64
	gasPriceDiff uint64

	stopCh chan struct{}
}

// NewLayer1Relayer will return a new instance of Layer1RelayerClient
func NewLayer1Relayer(ctx context.Context, db database.OrmFactory, cfg *config.RelayerConfig) (*Layer1Relayer, error) {
	messageSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.MessageSenderPrivateKeys)
	if err != nil {
		addr := crypto.PubkeyToAddress(cfg.MessageSenderPrivateKeys[0].PublicKey)
		log.Error("new MessageSender failed", "main address", addr.String(), "err", err)
		return nil, err
	}

	// @todo make sure only one sender is available
	gasOracleSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.GasOracleSenderPrivateKeys)
	if err != nil {
		addr := crypto.PubkeyToAddress(cfg.GasOracleSenderPrivateKeys[0].PublicKey)
		log.Error("new GasOracleSender failed", "main address", addr.String(), "err", err)
		return nil, err
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

	relayer := &Layer1Relayer{
		ctx: ctx,
		db:  db,

		messageSender:  messageSender,
		messageCh:      messageSender.ConfirmChan(),
		l2MessengerABI: bridge_abi.L2ScrollMessengerABI,

		gasOracleSender: gasOracleSender,
		gasOracleCh:     gasOracleSender.ConfirmChan(),
		l1GasOracleABI:  bridge_abi.L1GasPriceOracleABI,

		minGasPrice:  minGasPrice,
		gasPriceDiff: gasPriceDiff,

		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
	go relayer.confirmLoop(ctx)

	return relayer, nil
}

func (r *Layer1Relayer) prepare() error {
	if err := r.checkSubmittedMessages(); err != nil {
		log.Error("failed to init layer1 submitted tx", "err", err)
		return err
	}

	// Wait forever util sender is empty.
	utils.TryTimes(-1, func() bool {
		return r.messageSender.PendingCount() == 0
	})
	return nil
}

// Start the relayer process
func (r *Layer1Relayer) Start() {
	// Deal with broken transactions.
	if err := r.prepare(); err != nil {
		log.Crit("failed to init layer1 transaction messages")
	}

	loop := func(ctx context.Context, f func()) {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				f()
			}
		}
	}
	go func() {
		ctx, cancel := context.WithCancel(r.ctx)

		go loop(ctx, r.ProcessSavedEvents)
		go loop(ctx, r.ProcessGasPriceOracle)

		<-r.stopCh
		cancel()
	}()
}

func (r *Layer1Relayer) confirmLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cfm := <-r.messageCh:
			if !cfm.IsSuccessful {
				log.Warn("transaction confirmed but failed in layer2", "confirmation", cfm)
			} else {
				// @todo handle db error
				err := r.db.UpdateLayer1StatusAndLayer2Hash(r.ctx, cfm.ID, types.MsgConfirmed, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateLayer1StatusAndLayer2Hash failed", "err", err)
				}
				log.Info("transaction confirmed in layer2", "confirmation", cfm)
			}
		case cfm := <-r.gasOracleCh:
			if !cfm.IsSuccessful {
				// @discuss: maybe make it pending again?
				err := r.db.UpdateL1GasOracleStatusAndOracleTxHash(r.ctx, cfm.ID, types.GasOracleFailed, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateL1GasOracleStatusAndOracleTxHash failed", "err", err)
				}
				log.Warn("transaction confirmed but failed in layer2", "confirmation", cfm)
			} else {
				// @todo handle db error
				err := r.db.UpdateL1GasOracleStatusAndOracleTxHash(r.ctx, cfm.ID, types.GasOracleImported, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateGasOracleStatusAndOracleTxHash failed", "err", err)
				}
				log.Info("transaction confirmed in layer2", "confirmation", cfm)
			}
		}
	}
}

// Stop the relayer module, for a graceful shutdown.
func (r *Layer1Relayer) Stop() {
	close(r.stopCh)
}
