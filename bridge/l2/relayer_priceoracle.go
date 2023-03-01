package l2

import (
	"errors"
	"github.com/scroll-tech/go-ethereum/log"
	"math/big"
	"scroll-tech/bridge/sender"
	"scroll-tech/common/types"
)

// ProcessGasPriceOracle imports gas price to layer1
func (r *Layer2Relayer) ProcessGasPriceOracle() {
	batch, err := r.db.GetLatestBatch()
	if err != nil {
		log.Error("Failed to GetLatestBatch", "err", err)
		return
	}

	if batch.OracleStatus == types.GasOraclePending {
		suggestGasPrice, err := r.l2Client.SuggestGasPrice(r.ctx)
		if err != nil {
			log.Error("Failed to fetch SuggestGasPrice from l2geth", "err", err)
			return
		}
		suggestGasPriceUint64 := uint64(suggestGasPrice.Int64())
		expectedDelta := r.lastGasPrice * r.gasPriceDiff / gasPriceDiffPrecision

		// last is undefine or (suggestGasPriceUint64 >= minGasPrice && exceed diff)
		if r.lastGasPrice == 0 || (suggestGasPriceUint64 >= r.minGasPrice && (suggestGasPriceUint64 >= r.lastGasPrice+expectedDelta || suggestGasPriceUint64 <= r.lastGasPrice-expectedDelta)) {
			data, err := r.l2GasOracleABI.Pack("setL2BaseFee", suggestGasPrice)
			if err != nil {
				log.Error("Failed to pack setL2BaseFee", "batch.Hash", batch.Hash, "GasPrice", suggestGasPrice.Uint64(), "err", err)
				return
			}

			hash, err := r.gasOracleSender.SendTransaction(batch.Hash, &r.cfg.GasPriceOracleContractAddress, big.NewInt(0), data)
			if err != nil {
				if !errors.Is(err, sender.ErrNoAvailableAccount) {
					log.Error("Failed to send setL2BaseFee tx to layer2 ", "batch.Hash", batch.Hash, "err", err)
				}
				return
			}

			err = r.db.UpdateL2GasOracleStatusAndOracleTxHash(r.ctx, batch.Hash, types.GasOracleImporting, hash.String())
			if err != nil {
				log.Error("UpdateGasOracleStatusAndOracleTxHash failed", "batch.Hash", batch.Hash, "err", err)
				return
			}
			r.lastGasPrice = suggestGasPriceUint64
			log.Info("Update l2 gas price", "txHash", hash.String(), "GasPrice", suggestGasPrice)
		}
	}
}
