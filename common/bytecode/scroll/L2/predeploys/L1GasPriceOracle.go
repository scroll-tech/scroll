// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package predeploys

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = bind.Bind
	_ = common.Big1
	_ = abi.MakeTopics
)

// L1GasPriceOracleMetaData contains all meta data concerning the L1GasPriceOracle contract.
var (
	L1GasPriceOracleMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"l1BaseFee\",\"type\":\"uint256\"}],\"name\":\"L1BaseFeeUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"overhead\",\"type\":\"uint256\"}],\"name\":\"OverheadUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_oldOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"scalar\",\"type\":\"uint256\"}],\"name\":\"ScalarUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_oldWhitelist\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\"}],\"name\":\"UpdateWhitelist\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"getL1Fee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"getL1GasUsed\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1BaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"overhead\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"scalar\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l1BaseFee\",\"type\":\"uint256\"}],\"name\":\"setL1BaseFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_overhead\",\"type\":\"uint256\"}],\"name\":\"setOverhead\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_scalar\",\"type\":\"uint256\"}],\"name\":\"setScalar\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\"}],\"name\":\"updateWhitelist\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"whitelist\",\"outputs\":[{\"internalType\":\"contractIWhitelist\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	}
	// L1GasPriceOracleABI is the input ABI used to generate the binding from.
	L1GasPriceOracleABI *abi.ABI

	// L1BaseFeeUpdated event
	L1GasPriceOracleL1BaseFeeUpdatedEventSignature common.Hash

	// OverheadUpdated event
	L1GasPriceOracleOverheadUpdatedEventSignature common.Hash

	// OwnershipTransferred event
	L1GasPriceOracleOwnershipTransferredEventSignature common.Hash

	// ScalarUpdated event
	L1GasPriceOracleScalarUpdatedEventSignature common.Hash

	// UpdateWhitelist event
	L1GasPriceOracleUpdateWhitelistEventSignature common.Hash
)

func init() {
	sigAbi, err := L1GasPriceOracleMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	L1GasPriceOracleABI = sigAbi

	// L1BaseFeeUpdated event
	L1GasPriceOracleL1BaseFeeUpdatedEventSignature = sigAbi.Events["L1BaseFeeUpdated"].ID

	// OverheadUpdated event
	L1GasPriceOracleOverheadUpdatedEventSignature = sigAbi.Events["OverheadUpdated"].ID

	// OwnershipTransferred event
	L1GasPriceOracleOwnershipTransferredEventSignature = sigAbi.Events["OwnershipTransferred"].ID

	// ScalarUpdated event
	L1GasPriceOracleScalarUpdatedEventSignature = sigAbi.Events["ScalarUpdated"].ID

	// UpdateWhitelist event
	L1GasPriceOracleUpdateWhitelistEventSignature = sigAbi.Events["UpdateWhitelist"].ID

}

// L1BaseFeeUpdated represents a L1BaseFeeUpdated event raised by the L1GasPriceOracle contract.
type L1GasPriceOracleL1BaseFeeUpdatedEvent struct {
	L1BaseFee *big.Int
}

// OverheadUpdated represents a OverheadUpdated event raised by the L1GasPriceOracle contract.
type L1GasPriceOracleOverheadUpdatedEvent struct {
	Overhead *big.Int
}

// OwnershipTransferred represents a OwnershipTransferred event raised by the L1GasPriceOracle contract.
type L1GasPriceOracleOwnershipTransferredEvent struct {
	OldOwner common.Address
	NewOwner common.Address
}

// ScalarUpdated represents a ScalarUpdated event raised by the L1GasPriceOracle contract.
type L1GasPriceOracleScalarUpdatedEvent struct {
	Scalar *big.Int
}

// UpdateWhitelist represents a UpdateWhitelist event raised by the L1GasPriceOracle contract.
type L1GasPriceOracleUpdateWhitelistEvent struct {
	OldWhitelist common.Address
	NewWhitelist common.Address
}
