// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package predeploys

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

// L1GasPriceOracleMetaData contains all meta data concerning the L1GasPriceOracle contract.
var (
	L1GasPriceOracleMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"l1BaseFee\",\"type\":\"uint256\",\"indexed\":false}],\"type\":\"event\",\"name\":\"L1BaseFeeUpdated\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"overhead\",\"type\":\"uint256\",\"indexed\":false}],\"type\":\"event\",\"name\":\"OverheadUpdated\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oldOwner\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"scalar\",\"type\":\"uint256\",\"indexed\":false}],\"type\":\"event\",\"name\":\"ScalarUpdated\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oldWhitelist\",\"type\":\"address\",\"indexed\":false},{\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\",\"indexed\":false}],\"type\":\"event\",\"name\":\"UpdateWhitelist\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"getL1Fee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"getL1GasUsed\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"l1BaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"overhead\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"renounceOwnership\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"scalar\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l1BaseFee\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setL1BaseFee\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_overhead\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setOverhead\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_scalar\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setScalar\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"transferOwnership\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"updateWhitelist\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"whitelist\",\"outputs\":[{\"internalType\":\"contractIWhitelist\",\"name\":\"\",\"type\":\"address\"}]}]",
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

// L1GasPriceOracle is an auto generated Go binding around an Ethereum contract.
type L1GasPriceOracle struct {
	Address common.Address // contract address
	ABI     *abi.ABI       // L1GasPriceOracleABI is the input ABI used to generate the binding from.
	Parsers map[common.Hash]func(log *types.Log) (interface{}, error)

	L1GasPriceOracleCaller     // Read-only binding to the contract
	L1GasPriceOracleTransactor // Write-only binding to the contract
}

// GetAddress return L1GasPriceOracle's contract address.
func (o *L1GasPriceOracle) GetAddress() common.Address {
	return o.Address
}

// GetParsers return Parsers
func (o *L1GasPriceOracle) GetParsers() map[common.Hash]func(log *types.Log) (interface{}, error) {
	return o.Parsers
}

// GetABI return *big.ABI
func (o *L1GasPriceOracle) GetABI() *abi.ABI {
	return o.ABI
}

// L1GasPriceOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1GasPriceOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1GasPriceOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1GasPriceOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NewL1GasPriceOracle creates a new instance of L1GasPriceOracle, bound to a specific deployed contract.
func NewL1GasPriceOracle(address common.Address, backend bind.ContractBackend) (*L1GasPriceOracle, error) {
	contract, err := bindL1GasPriceOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	sigAbi, err := L1GasPriceOracleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	var parsers = map[common.Hash]func(log *types.Log) (interface{}, error){}

	parsers[sigAbi.Events["L1BaseFeeUpdated"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GasPriceOracleL1BaseFeeUpdatedEvent)
		if err := contract.UnpackLog(event, "L1BaseFeeUpdated", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["OverheadUpdated"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GasPriceOracleOverheadUpdatedEvent)
		if err := contract.UnpackLog(event, "OverheadUpdated", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["OwnershipTransferred"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GasPriceOracleOwnershipTransferredEvent)
		if err := contract.UnpackLog(event, "OwnershipTransferred", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["ScalarUpdated"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GasPriceOracleScalarUpdatedEvent)
		if err := contract.UnpackLog(event, "ScalarUpdated", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["UpdateWhitelist"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GasPriceOracleUpdateWhitelistEvent)
		if err := contract.UnpackLog(event, "UpdateWhitelist", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	return &L1GasPriceOracle{ABI: sigAbi, Address: address, Parsers: parsers, L1GasPriceOracleCaller: L1GasPriceOracleCaller{contract: contract}, L1GasPriceOracleTransactor: L1GasPriceOracleTransactor{contract: contract}}, nil
}

// NewL1GasPriceOracleCaller creates a new read-only instance of L1GasPriceOracle, bound to a specific deployed contract.
func NewL1GasPriceOracleCaller(address common.Address, caller bind.ContractCaller) (*L1GasPriceOracleCaller, error) {
	contract, err := bindL1GasPriceOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1GasPriceOracleCaller{contract: contract}, nil
}

// NewL1GasPriceOracleTransactor creates a new write-only instance of L1GasPriceOracle, bound to a specific deployed contract.
func NewL1GasPriceOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*L1GasPriceOracleTransactor, error) {
	contract, err := bindL1GasPriceOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1GasPriceOracleTransactor{contract: contract}, nil
}

// bindL1GasPriceOracle binds a generic wrapper to an already deployed contract.
func bindL1GasPriceOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1GasPriceOracleMetaData.ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// GetL1Fee is a free data retrieval call binding the contract method 0x49948e0e.
//
// Solidity: function getL1Fee(bytes _data) view returns(uint256)
func (_L1GasPriceOracle *L1GasPriceOracleCaller) GetL1Fee(opts *bind.CallOpts, _data []byte) (*big.Int, error) {
	var out []interface{}
	err := _L1GasPriceOracle.contract.Call(opts, &out, "getL1Fee", _data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1GasUsed is a free data retrieval call binding the contract method 0xde26c4a1.
//
// Solidity: function getL1GasUsed(bytes _data) view returns(uint256)
func (_L1GasPriceOracle *L1GasPriceOracleCaller) GetL1GasUsed(opts *bind.CallOpts, _data []byte) (*big.Int, error) {
	var out []interface{}
	err := _L1GasPriceOracle.contract.Call(opts, &out, "getL1GasUsed", _data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L1BaseFee is a free data retrieval call binding the contract method 0x519b4bd3.
//
// Solidity: function l1BaseFee() view returns(uint256)
func (_L1GasPriceOracle *L1GasPriceOracleCaller) L1BaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1GasPriceOracle.contract.Call(opts, &out, "l1BaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_L1GasPriceOracle *L1GasPriceOracleCaller) Overhead(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1GasPriceOracle.contract.Call(opts, &out, "overhead")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1GasPriceOracle *L1GasPriceOracleCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1GasPriceOracle.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_L1GasPriceOracle *L1GasPriceOracleCaller) Scalar(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1GasPriceOracle.contract.Call(opts, &out, "scalar")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Whitelist is a free data retrieval call binding the contract method 0x93e59dc1.
//
// Solidity: function whitelist() view returns(address)
func (_L1GasPriceOracle *L1GasPriceOracleCaller) Whitelist(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1GasPriceOracle.contract.Call(opts, &out, "whitelist")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1GasPriceOracle *L1GasPriceOracleTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1GasPriceOracle.contract.Transact(opts, "renounceOwnership")
}

// SetL1BaseFee is a paid mutator transaction binding the contract method 0xbede39b5.
//
// Solidity: function setL1BaseFee(uint256 _l1BaseFee) returns()
func (_L1GasPriceOracle *L1GasPriceOracleTransactor) SetL1BaseFee(opts *bind.TransactOpts, _l1BaseFee *big.Int) (*types.Transaction, error) {
	return _L1GasPriceOracle.contract.Transact(opts, "setL1BaseFee", _l1BaseFee)
}

// SetOverhead is a paid mutator transaction binding the contract method 0x3577afc5.
//
// Solidity: function setOverhead(uint256 _overhead) returns()
func (_L1GasPriceOracle *L1GasPriceOracleTransactor) SetOverhead(opts *bind.TransactOpts, _overhead *big.Int) (*types.Transaction, error) {
	return _L1GasPriceOracle.contract.Transact(opts, "setOverhead", _overhead)
}

// SetScalar is a paid mutator transaction binding the contract method 0x70465597.
//
// Solidity: function setScalar(uint256 _scalar) returns()
func (_L1GasPriceOracle *L1GasPriceOracleTransactor) SetScalar(opts *bind.TransactOpts, _scalar *big.Int) (*types.Transaction, error) {
	return _L1GasPriceOracle.contract.Transact(opts, "setScalar", _scalar)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_L1GasPriceOracle *L1GasPriceOracleTransactor) TransferOwnership(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _L1GasPriceOracle.contract.Transact(opts, "transferOwnership", _newOwner)
}

// UpdateWhitelist is a paid mutator transaction binding the contract method 0x3d0f963e.
//
// Solidity: function updateWhitelist(address _newWhitelist) returns()
func (_L1GasPriceOracle *L1GasPriceOracleTransactor) UpdateWhitelist(opts *bind.TransactOpts, _newWhitelist common.Address) (*types.Transaction, error) {
	return _L1GasPriceOracle.contract.Transact(opts, "updateWhitelist", _newWhitelist)
}

// L1GasPriceOracleL1BaseFeeUpdated represents a L1BaseFeeUpdated event raised by the L1GasPriceOracle contract.
type L1GasPriceOracleL1BaseFeeUpdatedEvent struct {
	L1BaseFee *big.Int
	raw       *types.Log // Blockchain specific contextual infos
}

// L1GasPriceOracleOverheadUpdated represents a OverheadUpdated event raised by the L1GasPriceOracle contract.
type L1GasPriceOracleOverheadUpdatedEvent struct {
	Overhead *big.Int
	raw      *types.Log // Blockchain specific contextual infos
}

// L1GasPriceOracleOwnershipTransferred represents a OwnershipTransferred event raised by the L1GasPriceOracle contract.
type L1GasPriceOracleOwnershipTransferredEvent struct {
	OldOwner common.Address
	NewOwner common.Address
	raw      *types.Log // Blockchain specific contextual infos
}

// L1GasPriceOracleScalarUpdated represents a ScalarUpdated event raised by the L1GasPriceOracle contract.
type L1GasPriceOracleScalarUpdatedEvent struct {
	Scalar *big.Int
	raw    *types.Log // Blockchain specific contextual infos
}

// L1GasPriceOracleUpdateWhitelist represents a UpdateWhitelist event raised by the L1GasPriceOracle contract.
type L1GasPriceOracleUpdateWhitelistEvent struct {
	OldWhitelist common.Address
	NewWhitelist common.Address
	raw          *types.Log // Blockchain specific contextual infos
}
