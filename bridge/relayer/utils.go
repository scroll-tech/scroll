package relayer

import "scroll-tech/bridge/sender"

const (
	gasPriceDiffPrecision = 1000000

	defaultGasPriceDiff = 50000 // 5%

	defaultL1MessageRelayMinGasLimit = 130000 // should be enough for both ERC20 and ETH relay

	defaultL2MessageRelayMinGasLimit = 200000
)

// ConfirmChs collects all chanels used in l1/l2 relayer
type ConfirmChs struct {
	messageCh   <-chan *sender.Confirmation
	gasOracleCh <-chan *sender.Confirmation
	rollupCh    <-chan *sender.Confirmation
}

// GetMsgChanel returns relayer's msg chanel
func (r *ConfirmChs) GetMsgChanel() <-chan *sender.Confirmation {
	return r.messageCh
}

// GetGasOracleChanel returns relayer's gas oracle chanel
func (r *ConfirmChs) GetGasOracleChanel() <-chan *sender.Confirmation {
	return r.gasOracleCh
}

// GetRollupChanel returns relayer's gas oracle chanel
func (r *ConfirmChs) GetRollupChanel() <-chan *sender.Confirmation {
	return r.rollupCh
}
