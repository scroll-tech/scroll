package relayer

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	bridgeAbi "scroll-tech/rollup/abi"
	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/sender"
	"scroll-tech/rollup/internal/orm"
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

	gasOracleSender *sender.Sender
	l1GasOracleABI  *abi.ABI

	lastGasPrice uint64
	minGasPrice  uint64
	gasPriceDiff uint64

	l1BlockOrm *orm.L1Block
	metrics    *l1RelayerMetrics
}

// NewLayer1Relayer will return a new instance of Layer1RelayerClient
func NewLayer1Relayer(ctx context.Context, db *gorm.DB, cfg *config.RelayerConfig, reg prometheus.Registerer) (*Layer1Relayer, error) {
	gasOracleSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.GasOracleSenderPrivateKey, "l1_relayer", "gas_oracle_sender", reg)
	if err != nil {
		addr := crypto.PubkeyToAddress(cfg.GasOracleSenderPrivateKey.PublicKey)
		return nil, fmt.Errorf("new gas oracle sender failed for address %s, err: %v", addr.Hex(), err)
	}

	// Ensure test features aren't enabled on the mainnet.
	if gasOracleSender.GetChainID() == big.NewInt(1) && cfg.EnableTestEnvBypassFeatures {
		return nil, fmt.Errorf("cannot enable test env features in mainnet")
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

	l1Relayer := &Layer1Relayer{
		cfg:        cfg,
		ctx:        ctx,
		l1BlockOrm: orm.NewL1Block(db),

		gasOracleSender: gasOracleSender,
		l1GasOracleABI:  bridgeAbi.L1GasPriceOracleABI,

		minGasPrice:  minGasPrice,
		gasPriceDiff: gasPriceDiff,
	}

	l1Relayer.metrics = initL1RelayerMetrics(reg)

	go l1Relayer.handleConfirmLoop(ctx)
	return l1Relayer, nil
}

// ProcessGasPriceOracle imports gas price to layer2
func (r *Layer1Relayer) ProcessGasPriceOracle() {
	r.metrics.rollupL1RelayerGasPriceOraclerRunTotal.Inc()
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
			r.metrics.rollupL1RelayerLastGasPrice.Set(float64(r.lastGasPrice))
			log.Info("Update l1 base fee", "txHash", hash.String(), "baseFee", baseFee)
		}
	}
}

func (r *Layer1Relayer) handleConfirmLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cfm := <-r.gasOracleSender.ConfirmChan():
			r.metrics.rollupL1GasOraclerConfirmedTotal.Inc()
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
