// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package rollup

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// L2GasPriceOracleMetaData contains all meta data concerning the L2GasPriceOracle contract.
var (
	L2GasPriceOracleMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Initialized\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"txGas\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"txGasContractCreation\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"zeroGas\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"nonZeroGas\",\"type\":\"uint256\",\"indexed\":false}],\"type\":\"event\",\"name\":\"IntrinsicParamsUpdated\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"oldL2BaseFee\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"newL2BaseFee\",\"type\":\"uint256\",\"indexed\":false}],\"type\":\"event\",\"name\":\"L2BaseFeeUpdated\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oldWhitelist\",\"type\":\"address\",\"indexed\":false},{\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\",\"indexed\":false}],\"type\":\"event\",\"name\":\"UpdateWhitelist\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"calculateIntrinsicGasFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"estimateCrossDomainMessageFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_txGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_txGasContractCreation\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_zeroGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_nonZeroGas\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"initialize\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"intrinsicParams\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"txGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"txGasContractCreation\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"zeroGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"nonZeroGas\",\"type\":\"uint64\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"l2BaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"renounceOwnership\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_txGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_txGasContractCreation\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_zeroGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_nonZeroGas\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setIntrinsicParams\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_newL2BaseFee\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setL2BaseFee\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"transferOwnership\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"updateWhitelist\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"whitelist\",\"outputs\":[{\"internalType\":\"contractIWhitelist\",\"name\":\"\",\"type\":\"address\"}]}]",
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

// L2GasPriceOracle is an auto generated Go binding around an Ethereum contract.
type L2GasPriceOracle struct {
	Address common.Address // contract address
	ABI     *abi.ABI       // L2GasPriceOracleABI is the input ABI used to generate the binding from.
	Parsers map[common.Hash]func(log *types.Log) (interface{}, error)

	L2GasPriceOracleCaller     // Read-only binding to the contract
	L2GasPriceOracleTransactor // Write-only binding to the contract
}

// GetAddress return L2GasPriceOracle's contract address.
func (o *L2GasPriceOracle) GetAddress() common.Address {
	return o.Address
}

// GetParsers return Parsers
func (o *L2GasPriceOracle) GetParsers() map[common.Hash]func(log *types.Log) (interface{}, error) {
	return o.Parsers
}

// GetABI return *big.ABI
func (o *L2GasPriceOracle) GetABI() *abi.ABI {
	return o.ABI
}

// L2GasPriceOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2GasPriceOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2GasPriceOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2GasPriceOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NewL2GasPriceOracle creates a new instance of L2GasPriceOracle, bound to a specific deployed contract.
func NewL2GasPriceOracle(address common.Address, backend bind.ContractBackend) (*L2GasPriceOracle, error) {
	contract, err := bindL2GasPriceOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	sigAbi, err := L2GasPriceOracleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	var parsers = map[common.Hash]func(log *types.Log) (interface{}, error){}

	parsers[sigAbi.Events["Initialized"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GasPriceOracleInitializedEvent)
		if err := contract.UnpackLog(event, "Initialized", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["IntrinsicParamsUpdated"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GasPriceOracleIntrinsicParamsUpdatedEvent)
		if err := contract.UnpackLog(event, "IntrinsicParamsUpdated", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["L2BaseFeeUpdated"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GasPriceOracleL2BaseFeeUpdatedEvent)
		if err := contract.UnpackLog(event, "L2BaseFeeUpdated", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["OwnershipTransferred"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GasPriceOracleOwnershipTransferredEvent)
		if err := contract.UnpackLog(event, "OwnershipTransferred", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["UpdateWhitelist"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GasPriceOracleUpdateWhitelistEvent)
		if err := contract.UnpackLog(event, "UpdateWhitelist", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	return &L2GasPriceOracle{ABI: sigAbi, Address: address, Parsers: parsers, L2GasPriceOracleCaller: L2GasPriceOracleCaller{contract: contract}, L2GasPriceOracleTransactor: L2GasPriceOracleTransactor{contract: contract}}, nil
}

// NewL2GasPriceOracleCaller creates a new read-only instance of L2GasPriceOracle, bound to a specific deployed contract.
func NewL2GasPriceOracleCaller(address common.Address, caller bind.ContractCaller) (*L2GasPriceOracleCaller, error) {
	contract, err := bindL2GasPriceOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2GasPriceOracleCaller{contract: contract}, nil
}

// NewL2GasPriceOracleTransactor creates a new write-only instance of L2GasPriceOracle, bound to a specific deployed contract.
func NewL2GasPriceOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*L2GasPriceOracleTransactor, error) {
	contract, err := bindL2GasPriceOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2GasPriceOracleTransactor{contract: contract}, nil
}

// bindL2GasPriceOracle binds a generic wrapper to an already deployed contract.
func bindL2GasPriceOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2GasPriceOracleMetaData.ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// CalculateIntrinsicGasFee is a free data retrieval call binding the contract method 0xe172d3a1.
//
// Solidity: function calculateIntrinsicGasFee(bytes _message) view returns(uint256)
func (_L2GasPriceOracle *L2GasPriceOracleCaller) CalculateIntrinsicGasFee(opts *bind.CallOpts, _message []byte) (*big.Int, error) {
	var out []interface{}
	err := _L2GasPriceOracle.contract.Call(opts, &out, "calculateIntrinsicGasFee", _message)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EstimateCrossDomainMessageFee is a free data retrieval call binding the contract method 0xd7704bae.
//
// Solidity: function estimateCrossDomainMessageFee(uint256 _gasLimit) view returns(uint256)
func (_L2GasPriceOracle *L2GasPriceOracleCaller) EstimateCrossDomainMessageFee(opts *bind.CallOpts, _gasLimit *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _L2GasPriceOracle.contract.Call(opts, &out, "estimateCrossDomainMessageFee", _gasLimit)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// IntrinsicParams is a free data retrieval call binding the contract method 0x64431a27.
//
// Solidity: function intrinsicParams() view returns(uint64 txGas, uint64 txGasContractCreation, uint64 zeroGas, uint64 nonZeroGas)
func (_L2GasPriceOracle *L2GasPriceOracleCaller) IntrinsicParams(opts *bind.CallOpts) (struct {
	TxGas                 uint64
	TxGasContractCreation uint64
	ZeroGas               uint64
	NonZeroGas            uint64
}, error) {
	var out []interface{}
	err := _L2GasPriceOracle.contract.Call(opts, &out, "intrinsicParams")

	outstruct := new(struct {
		TxGas                 uint64
		TxGasContractCreation uint64
		ZeroGas               uint64
		NonZeroGas            uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.TxGas = *abi.ConvertType(out[0], new(uint64)).(*uint64)
	outstruct.TxGasContractCreation = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.ZeroGas = *abi.ConvertType(out[2], new(uint64)).(*uint64)
	outstruct.NonZeroGas = *abi.ConvertType(out[3], new(uint64)).(*uint64)

	return *outstruct, err

}

// L2BaseFee is a free data retrieval call binding the contract method 0xe3176bd5.
//
// Solidity: function l2BaseFee() view returns(uint256)
func (_L2GasPriceOracle *L2GasPriceOracleCaller) L2BaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2GasPriceOracle.contract.Call(opts, &out, "l2BaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2GasPriceOracle *L2GasPriceOracleCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2GasPriceOracle.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Whitelist is a free data retrieval call binding the contract method 0x93e59dc1.
//
// Solidity: function whitelist() view returns(address)
func (_L2GasPriceOracle *L2GasPriceOracleCaller) Whitelist(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2GasPriceOracle.contract.Call(opts, &out, "whitelist")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Initialize is a paid mutator transaction binding the contract method 0x3366ff72.
//
// Solidity: function initialize(uint64 _txGas, uint64 _txGasContractCreation, uint64 _zeroGas, uint64 _nonZeroGas) returns()
func (_L2GasPriceOracle *L2GasPriceOracleTransactor) Initialize(opts *bind.TransactOpts, _txGas uint64, _txGasContractCreation uint64, _zeroGas uint64, _nonZeroGas uint64) (*types.Transaction, error) {
	return _L2GasPriceOracle.contract.Transact(opts, "initialize", _txGas, _txGasContractCreation, _zeroGas, _nonZeroGas)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2GasPriceOracle *L2GasPriceOracleTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2GasPriceOracle.contract.Transact(opts, "renounceOwnership")
}

// SetIntrinsicParams is a paid mutator transaction binding the contract method 0xaccf9a60.
//
// Solidity: function setIntrinsicParams(uint64 _txGas, uint64 _txGasContractCreation, uint64 _zeroGas, uint64 _nonZeroGas) returns()
func (_L2GasPriceOracle *L2GasPriceOracleTransactor) SetIntrinsicParams(opts *bind.TransactOpts, _txGas uint64, _txGasContractCreation uint64, _zeroGas uint64, _nonZeroGas uint64) (*types.Transaction, error) {
	return _L2GasPriceOracle.contract.Transact(opts, "setIntrinsicParams", _txGas, _txGasContractCreation, _zeroGas, _nonZeroGas)
}

// SetL2BaseFee is a paid mutator transaction binding the contract method 0xd99bc80e.
//
// Solidity: function setL2BaseFee(uint256 _newL2BaseFee) returns()
func (_L2GasPriceOracle *L2GasPriceOracleTransactor) SetL2BaseFee(opts *bind.TransactOpts, _newL2BaseFee *big.Int) (*types.Transaction, error) {
	return _L2GasPriceOracle.contract.Transact(opts, "setL2BaseFee", _newL2BaseFee)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2GasPriceOracle *L2GasPriceOracleTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L2GasPriceOracle.contract.Transact(opts, "transferOwnership", newOwner)
}

// UpdateWhitelist is a paid mutator transaction binding the contract method 0x3d0f963e.
//
// Solidity: function updateWhitelist(address _newWhitelist) returns()
func (_L2GasPriceOracle *L2GasPriceOracleTransactor) UpdateWhitelist(opts *bind.TransactOpts, _newWhitelist common.Address) (*types.Transaction, error) {
	return _L2GasPriceOracle.contract.Transact(opts, "updateWhitelist", _newWhitelist)
}

// L2GasPriceOracleInitialized represents a Initialized event raised by the L2GasPriceOracle contract.
type L2GasPriceOracleInitializedEvent struct {
	Version uint8
	raw     *types.Log // Blockchain specific contextual infos
}

// L2GasPriceOracleIntrinsicParamsUpdated represents a IntrinsicParamsUpdated event raised by the L2GasPriceOracle contract.
type L2GasPriceOracleIntrinsicParamsUpdatedEvent struct {
	TxGas                 *big.Int
	TxGasContractCreation *big.Int
	ZeroGas               *big.Int
	NonZeroGas            *big.Int
	raw                   *types.Log // Blockchain specific contextual infos
}

// L2GasPriceOracleL2BaseFeeUpdated represents a L2BaseFeeUpdated event raised by the L2GasPriceOracle contract.
type L2GasPriceOracleL2BaseFeeUpdatedEvent struct {
	OldL2BaseFee *big.Int
	NewL2BaseFee *big.Int
	raw          *types.Log // Blockchain specific contextual infos
}

// L2GasPriceOracleOwnershipTransferred represents a OwnershipTransferred event raised by the L2GasPriceOracle contract.
type L2GasPriceOracleOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	raw           *types.Log // Blockchain specific contextual infos
}

// L2GasPriceOracleUpdateWhitelist represents a UpdateWhitelist event raised by the L2GasPriceOracle contract.
type L2GasPriceOracleUpdateWhitelistEvent struct {
	OldWhitelist common.Address
	NewWhitelist common.Address
	raw          *types.Log // Blockchain specific contextual infos
}
