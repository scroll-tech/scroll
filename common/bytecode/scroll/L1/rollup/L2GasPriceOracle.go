// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package rollup

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

// L2GasPriceOracleMetaData contains all meta data concerning the L2GasPriceOracle contract.
var (
	L2GasPriceOracleMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"txGas\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"txGasContractCreation\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"zeroGas\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonZeroGas\",\"type\":\"uint256\"}],\"name\":\"IntrinsicParamsUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"l2BaseFee\",\"type\":\"uint256\"}],\"name\":\"L2BaseFeeUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_oldWhitelist\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\"}],\"name\":\"UpdateWhitelist\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"name\":\"calculateIntrinsicGasFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"estimateCrossDomainMessageFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_txGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_txGasContractCreation\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_zeroGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_nonZeroGas\",\"type\":\"uint64\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"intrinsicParams\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"txGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"txGasContractCreation\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"zeroGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"nonZeroGas\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2BaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_txGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_txGasContractCreation\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_zeroGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_nonZeroGas\",\"type\":\"uint64\"}],\"name\":\"setIntrinsicParams\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BaseFee\",\"type\":\"uint256\"}],\"name\":\"setL2BaseFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\"}],\"name\":\"updateWhitelist\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"whitelist\",\"outputs\":[{\"internalType\":\"contractIWhitelist\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	}
	// L2GasPriceOracleABI is the input ABI used to generate the binding from.
	L2GasPriceOracleABI *abi.ABI

	// Initialized event
	L2GasPriceOracleInitializedEventSignature common.Hash

	// IntrinsicParamsUpdated event
	L2GasPriceOracleIntrinsicParamsUpdatedEventSignature common.Hash

	// L2BaseFeeUpdated event
	L2GasPriceOracleL2BaseFeeUpdatedEventSignature common.Hash

	// OwnershipTransferred event
	L2GasPriceOracleOwnershipTransferredEventSignature common.Hash

	// UpdateWhitelist event
	L2GasPriceOracleUpdateWhitelistEventSignature common.Hash
)

func init() {
	sigAbi, err := L2GasPriceOracleMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	L2GasPriceOracleABI = sigAbi

	// Initialized event
	L2GasPriceOracleInitializedEventSignature = sigAbi.Events["Initialized"].ID

	// IntrinsicParamsUpdated event
	L2GasPriceOracleIntrinsicParamsUpdatedEventSignature = sigAbi.Events["IntrinsicParamsUpdated"].ID

	// L2BaseFeeUpdated event
	L2GasPriceOracleL2BaseFeeUpdatedEventSignature = sigAbi.Events["L2BaseFeeUpdated"].ID

	// OwnershipTransferred event
	L2GasPriceOracleOwnershipTransferredEventSignature = sigAbi.Events["OwnershipTransferred"].ID

	// UpdateWhitelist event
	L2GasPriceOracleUpdateWhitelistEventSignature = sigAbi.Events["UpdateWhitelist"].ID

}

// Initialized represents a Initialized event raised by the L2GasPriceOracle contract.
type L2GasPriceOracleInitializedEvent struct {
	Version uint8
}

// IntrinsicParamsUpdated represents a IntrinsicParamsUpdated event raised by the L2GasPriceOracle contract.
type L2GasPriceOracleIntrinsicParamsUpdatedEvent struct {
	TxGas                 *big.Int
	TxGasContractCreation *big.Int
	ZeroGas               *big.Int
	NonZeroGas            *big.Int
}

// L2BaseFeeUpdated represents a L2BaseFeeUpdated event raised by the L2GasPriceOracle contract.
type L2GasPriceOracleL2BaseFeeUpdatedEvent struct {
	L2BaseFee *big.Int
}

// OwnershipTransferred represents a OwnershipTransferred event raised by the L2GasPriceOracle contract.
type L2GasPriceOracleOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
}

// UpdateWhitelist represents a UpdateWhitelist event raised by the L2GasPriceOracle contract.
type L2GasPriceOracleUpdateWhitelistEvent struct {
	OldWhitelist common.Address
	NewWhitelist common.Address
}
