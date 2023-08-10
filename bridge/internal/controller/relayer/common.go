package relayer

import "errors"

const (
	gasPriceDiffPrecision = 1000000

	defaultGasPriceDiff = 50000 // 5%
)

var (
	// ErrExecutionRevertedMessageExpired error of Message expired
	ErrExecutionRevertedMessageExpired = errors.New("execution reverted: Message expired")
	// ErrExecutionRevertedAlreadySuccessExecuted error of Message was already successfully executed
	ErrExecutionRevertedAlreadySuccessExecuted = errors.New("execution reverted: Message was already successfully executed")
)
