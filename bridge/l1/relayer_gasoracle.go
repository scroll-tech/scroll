package l1

import (
	"errors"
	"math/big"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"

	"scroll-tech/bridge/sender"
)

// ProcessGasPriceOracle imports gas price to layer2
func (r *Layer1Relayer) ProcessGasPriceOracle() {
	latestBlockHeight, err := r.db.GetLatestL1BlockHeight()
	if err != nil {
		log.Warn("Failed to fetch latest L1 block height from db", "err", err)
		return
	}

	blocks, err := r.db.GetL1BlockInfos(map[string]interface{}{
		"number": latestBlockHeight,
	})
	if err != nil {
		log.Error("Failed to GetL1BlockInfos from db", "height", latestBlockHeight, "err", err)
		return
	}
	if len(blocks) != 1 {
		log.Error("Block not exist", "height", latestBlockHeight)
		return
	}
	block := blocks[0]

	if block.GasOracleStatus == types.GasOraclePending {
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
				if !errors.Is(err, sender.ErrNoAvailableAccount) {
					log.Error("Failed to send setL1BaseFee tx to layer2 ", "block.Hash", block.Hash, "block.Height", block.Number, "err", err)
				}
				return
			}

			err = r.db.UpdateL1GasOracleStatusAndOracleTxHash(r.ctx, block.Hash, types.GasOracleImporting, hash.String())
			if err != nil {
				log.Error("UpdateGasOracleStatusAndOracleTxHash failed", "block.Hash", block.Hash, "block.Height", block.Number, "err", err)
				return
			}
			r.lastGasPrice = block.BaseFee
			log.Info("Update l1 base fee", "txHash", hash.String(), "baseFee", baseFee)
		}
	}
}
