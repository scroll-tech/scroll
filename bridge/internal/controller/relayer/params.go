package relayer

import "errors"

const (
	gasPriceDiffPrecision = 1000000

	defaultGasPriceDiff = 50000 // 5%

	defaultL1MessageRelayMinGasLimit = 130000 // should be enough for both ERC20 and ETH relay

	defaultL2MessageRelayMinGasLimit = 200000
)

var (
	ErrExecutionRevertedMessageExpired         = errors.New("execution reverted: Message expired")
	ErrExecutionRevertedAlreadySuccessExecuted = errors.New("execution reverted: Message was already successfully executed")
)
