package relayer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/utils"

	bridgeAbi "scroll-tech/rollup/abi"
	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/sender"
	"scroll-tech/rollup/internal/orm"
	rutils "scroll-tech/rollup/internal/utils"
)

// Layer1Relayer is responsible for updating L1 gas price oracle contract on L2.
//
// Actions are triggered by L1 watcher.
type Layer1Relayer struct {
	ctx context.Context

	cfg *config.RelayerConfig

	gasOracleSender *sender.Sender
	l1GasOracleABI  *abi.ABI

	lastBaseFee     uint64
	lastBlobBaseFee uint64
	minGasPrice     uint64
	gasPriceDiff    uint64

	l1BlockOrm *orm.L1Block
	l2BlockOrm *orm.L2Block
	batchOrm   *orm.Batch

	metrics *l1RelayerMetrics
}

// NewLayer1Relayer will return a new instance of Layer1RelayerClient
func NewLayer1Relayer(ctx context.Context, db *gorm.DB, cfg *config.RelayerConfig, serviceType ServiceType, reg prometheus.Registerer) (*Layer1Relayer, error) {
	var gasOracleSender *sender.Sender
	var err error

	switch serviceType {
	case ServiceTypeL1GasOracle:
		gasOracleSender, err = sender.NewSender(ctx, cfg.SenderConfig, cfg.GasOracleSenderSignerConfig, "l1_relayer", "gas_oracle_sender", types.SenderTypeL1GasOracle, db, reg)
		if err != nil {
			return nil, fmt.Errorf("new gas oracle sender failed, err: %w", err)
		}

		// Ensure test features aren't enabled on the scroll mainnet.
		if gasOracleSender.GetChainID().Cmp(big.NewInt(534352)) == 0 && cfg.EnableTestEnvBypassFeatures {
			return nil, errors.New("cannot enable test env features in mainnet")
		}
	default:
		return nil, fmt.Errorf("invalid service type for l1_relayer: %v", serviceType)
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
		l2BlockOrm: orm.NewL2Block(db),
		batchOrm:   orm.NewBatch(db),

		gasOracleSender: gasOracleSender,
		l1GasOracleABI:  bridgeAbi.L1GasPriceOracleABI,

		minGasPrice:  minGasPrice,
		gasPriceDiff: gasPriceDiff,
	}

	l1Relayer.metrics = initL1RelayerMetrics(reg)

	switch serviceType {
	case ServiceTypeL1GasOracle:
		go l1Relayer.handleL1GasOracleConfirmLoop(ctx)
	default:
		return nil, fmt.Errorf("invalid service type for l1_relayer: %v", serviceType)
	}

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
		if block.BaseFee == 0 || block.BlobBaseFee == 0 {
			log.Error("Invalid base fee or blob base fee", "block.Hash", block.Hash, "block.Height", block.Number, "block.BaseFee", block.BaseFee, "block.BlobBaseFee", block.BlobBaseFee)
			return
		}

		baseFee := block.BaseFee
		blobBaseFee := block.BlobBaseFee

		// include the token exchange rate in the fee data if alternative gas token enabled
		if r.cfg.GasOracleConfig.AlternativeGasTokenConfig != nil && r.cfg.GasOracleConfig.AlternativeGasTokenConfig.Enabled {
			// The exchange rate represent the number of native token on L1 required to exchange for 1 native token on L2.
			var exchangeRate float64
			switch r.cfg.GasOracleConfig.AlternativeGasTokenConfig.Mode {
			case "Fixed":
				exchangeRate = r.cfg.GasOracleConfig.AlternativeGasTokenConfig.FixedExchangeRate
			case "BinanceApi":
				exchangeRate, err = rutils.GetExchangeRateFromBinanceApi(r.cfg.GasOracleConfig.AlternativeGasTokenConfig.TokenSymbolPair, 5)
				if err != nil {
					log.Error("Failed to get gas token exchange rate from Binance api", "tokenSymbolPair", r.cfg.GasOracleConfig.AlternativeGasTokenConfig.TokenSymbolPair, "err", err)
					return
				}
			default:
				log.Error("Invalid alternative gas token mode", "mode", r.cfg.GasOracleConfig.AlternativeGasTokenConfig.Mode)
				return
			}
			if exchangeRate == 0 {
				log.Error("Invalid exchange rate", "exchangeRate", exchangeRate)
				return
			}
			baseFee = uint64(math.Ceil(float64(baseFee) / exchangeRate))
			blobBaseFee = uint64(math.Ceil(float64(blobBaseFee) / exchangeRate))
		}

		if r.shouldUpdateGasOracle(baseFee, blobBaseFee) {
			// It indicates the committing batch has been stuck for a long time, it's likely that the L1 gas fee spiked.
			// If we are not committing batches due to high fees then we shouldn't update fees to prevent users from paying high l1_data_fee
			// Also, set fees to some default value, because we have already updated fees to some high values, probably
			var reachTimeout bool
			if reachTimeout, err = r.commitBatchReachTimeout(); reachTimeout && block.BlobBaseFee > r.cfg.GasOracleConfig.L1BlobBaseFeeThreshold && err == nil {
				if r.lastBaseFee == r.cfg.GasOracleConfig.L1BaseFeeDefault && r.lastBlobBaseFee == r.cfg.GasOracleConfig.L1BlobBaseFeeDefault {
					return
				}
				baseFee = r.cfg.GasOracleConfig.L1BaseFeeDefault
				blobBaseFee = r.cfg.GasOracleConfig.L1BlobBaseFeeDefault
			} else if err != nil {
				return
			}
			data, err := r.l1GasOracleABI.Pack("setL1BaseFeeAndBlobBaseFee", new(big.Int).SetUint64(baseFee), new(big.Int).SetUint64(blobBaseFee))
			if err != nil {
				log.Error("Failed to pack setL1BaseFeeAndBlobBaseFee", "block.Hash", block.Hash, "block.Height", block.Number, "block.BaseFee", baseFee, "block.BlobBaseFee", blobBaseFee, "err", err)
				return
			}

			hash, err := r.gasOracleSender.SendTransaction(block.Hash, &r.cfg.GasPriceOracleContractAddress, data, nil, 0)
			if err != nil {
				log.Error("Failed to send gas oracle update tx to layer2", "block.Hash", block.Hash, "block.Height", block.Number, "block.BaseFee", baseFee, "block.BlobBaseFee", blobBaseFee, "err", err)
				return
			}

			err = r.l1BlockOrm.UpdateL1GasOracleStatusAndOracleTxHash(r.ctx, block.Hash, types.GasOracleImporting, hash.String())
			if err != nil {
				log.Error("UpdateGasOracleStatusAndOracleTxHash failed", "block.Hash", block.Hash, "block.Height", block.Number, "err", err)
				return
			}

			r.lastBaseFee = baseFee
			r.lastBlobBaseFee = blobBaseFee
			r.metrics.rollupL1RelayerLatestBaseFee.Set(float64(r.lastBaseFee))
			r.metrics.rollupL1RelayerLatestBlobBaseFee.Set(float64(r.lastBlobBaseFee))
			log.Info("Update l1 base fee", "txHash", hash.String(), "baseFee", baseFee, "blobBaseFee", blobBaseFee)
		}
	}
}

func (r *Layer1Relayer) handleConfirmation(cfm *sender.Confirmation) {
	switch cfm.SenderType {
	case types.SenderTypeL1GasOracle:
		var status types.GasOracleStatus
		if cfm.IsSuccessful {
			status = types.GasOracleImported
			r.metrics.rollupL1UpdateGasOracleConfirmedTotal.Inc()
			log.Info("UpdateGasOracleTxType transaction confirmed in layer2", "confirmation", cfm)
		} else {
			status = types.GasOracleImportedFailed
			r.metrics.rollupL1UpdateGasOracleConfirmedFailedTotal.Inc()
			log.Warn("UpdateGasOracleTxType transaction confirmed but failed in layer2", "confirmation", cfm)
		}

		err := r.l1BlockOrm.UpdateL1GasOracleStatusAndOracleTxHash(r.ctx, cfm.ContextID, status, cfm.TxHash.String())
		if err != nil {
			log.Warn("UpdateL1GasOracleStatusAndOracleTxHash failed", "confirmation", cfm, "err", err)
		}
	default:
		log.Warn("Unknown transaction type", "confirmation", cfm)
	}

	log.Info("Transaction confirmed in layer2", "confirmation", cfm)
}

func (r *Layer1Relayer) handleL1GasOracleConfirmLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cfm := <-r.gasOracleSender.ConfirmChan():
			r.handleConfirmation(cfm)
		}
	}
}

// StopSenders stops the senders of the rollup-relayer to prevent querying the removed pending_transaction table in unit tests.
// for unit test
func (r *Layer1Relayer) StopSenders() {
	if r.gasOracleSender != nil {
		r.gasOracleSender.Stop()
	}
}

func (r *Layer1Relayer) shouldUpdateGasOracle(baseFee uint64, blobBaseFee uint64) bool {
	// Right after restarting.
	if r.lastBaseFee == 0 {
		log.Info("First time to update gas oracle after restarting", "baseFee", baseFee, "blobBaseFee", blobBaseFee)
		return true
	}

	expectedBaseFeeDelta := r.lastBaseFee*r.gasPriceDiff/gasPriceDiffPrecision + 1
	if baseFee >= r.minGasPrice && math.Abs(float64(baseFee)-float64(r.lastBaseFee)) >= float64(expectedBaseFeeDelta) {
		return true
	}

	expectedBlobBaseFeeDelta := r.lastBlobBaseFee * r.gasPriceDiff / gasPriceDiffPrecision
	// Plus a minimum of 0.01 gwei, since the blob base fee is usually low, preventing short-time flunctuation.
	expectedBlobBaseFeeDelta += 10000000
	if blobBaseFee >= r.minGasPrice && math.Abs(float64(blobBaseFee)-float64(r.lastBlobBaseFee)) >= float64(expectedBlobBaseFeeDelta) {
		return true
	}

	return false
}

func (r *Layer1Relayer) commitBatchReachTimeout() (bool, error) {
	fields := map[string]interface{}{
		"rollup_status IN ?": []types.RollupStatus{types.RollupCommitted, types.RollupFinalizing, types.RollupFinalized},
	}
	orderByList := []string{"index DESC"}
	limit := 1
	batches, err := r.batchOrm.GetBatches(r.ctx, fields, orderByList, limit)
	if err != nil {
		log.Warn("failed to fetch latest committed, finalizing or finalized batch", "err", err)
		return false, err
	}
	// len(batches) == 0 probably shouldn't ever happen, but need to check this
	// Also, we should check if it's a genesis batch. If so, skip the timeout check.
	return len(batches) == 0 || (batches[0].Index != 0 && utils.NowUTC().Sub(*batches[0].CommittedAt) > time.Duration(r.cfg.GasOracleConfig.CheckCommittedBatchesWindowMinutes)*time.Minute), nil
}
