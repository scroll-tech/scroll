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

// ServiceType defines the various types of services within the relayer.
type ServiceType int

const (
	// ServiceTypeUnknown indicates an unknown service type.
	ServiceTypeUnknown ServiceType = iota
	// ServiceTypeL2RollupRelayer indicates the service is a Layer 2 rollup relayer.
	ServiceTypeL2RollupRelayer
	// ServiceTypeL1GasOracle indicates the service is a Layer 1 gas oracle.
	ServiceTypeL1GasOracle
	// ServiceTypeL2GasOracle indicates the service is a Layer 2 gas oracle.
	ServiceTypeL2GasOracle
)
