package relayer

import "scroll-tech/bridge/sender"

const (
	gasPriceDiffPrecision = 1000000

	defaultGasPriceDiff = 50000 // 5%
)

type RelayerConfirmChs struct {
	messageCh   <-chan *sender.Confirmation
	gasOracleCh <-chan *sender.Confirmation
	rollupCh    <-chan *sender.Confirmation
}

// GetMsgChanel returns relayer's msg chanel
func (r *RelayerConfirmChs) GetMsgChanel() <-chan *sender.Confirmation {
	return r.messageCh
}

// GetGasOracleChanel returns relayer's gas oracle chanel
func (r *RelayerConfirmChs) GetGasOracleChanel() <-chan *sender.Confirmation {
	return r.gasOracleCh
}

// GetRollupChanel returns relayer's gas oracle chanel
func (r *RelayerConfirmChs) GetRollupChanel() <-chan *sender.Confirmation {
	return r.rollupCh
}
