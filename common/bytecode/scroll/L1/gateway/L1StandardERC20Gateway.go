// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package gateway

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

// L1StandardERC20GatewayMetaData contains all meta data concerning the L1StandardERC20Gateway contract.
var (
	L1StandardERC20GatewayMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"DepositERC20\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"FinalizeWithdrawERC20\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Initialized\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false}],\"type\":\"event\",\"name\":\"RefundERC20\",\"anonymous\":false},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"counterpart\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"depositERC20\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"depositERC20\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"depositERC20AndCall\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1Token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_l2Token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"finalizeWithdrawERC20\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1Token\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"getL2ERC20Address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_counterpart\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_router\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_messenger\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_l2TokenImplementation\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_l2TokenFactory\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"initialize\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"l2TokenFactory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"l2TokenImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"onDropMessage\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"renounceOwnership\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"router\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"transferOwnership\"}]",
	}
	// L1StandardERC20GatewayABI is the input ABI used to generate the binding from.
	L1StandardERC20GatewayABI *abi.ABI

	// DepositERC20 event
	L1StandardERC20GatewayDepositERC20EventSignature common.Hash

	// FinalizeWithdrawERC20 event
	L1StandardERC20GatewayFinalizeWithdrawERC20EventSignature common.Hash

	// Initialized event
	L1StandardERC20GatewayInitializedEventSignature common.Hash

	// OwnershipTransferred event
	L1StandardERC20GatewayOwnershipTransferredEventSignature common.Hash

	// RefundERC20 event
	L1StandardERC20GatewayRefundERC20EventSignature common.Hash
)

func init() {
	sigAbi, err := L1StandardERC20GatewayMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	L1StandardERC20GatewayABI = sigAbi

	// DepositERC20 event
	L1StandardERC20GatewayDepositERC20EventSignature = sigAbi.Events["DepositERC20"].ID

	// FinalizeWithdrawERC20 event
	L1StandardERC20GatewayFinalizeWithdrawERC20EventSignature = sigAbi.Events["FinalizeWithdrawERC20"].ID

	// Initialized event
	L1StandardERC20GatewayInitializedEventSignature = sigAbi.Events["Initialized"].ID

	// OwnershipTransferred event
	L1StandardERC20GatewayOwnershipTransferredEventSignature = sigAbi.Events["OwnershipTransferred"].ID

	// RefundERC20 event
	L1StandardERC20GatewayRefundERC20EventSignature = sigAbi.Events["RefundERC20"].ID

}

// L1StandardERC20Gateway is an auto generated Go binding around an Ethereum contract.
type L1StandardERC20Gateway struct {
	Address common.Address // contract address
	ABI     *abi.ABI       // L1StandardERC20GatewayABI is the input ABI used to generate the binding from.
	Parsers map[common.Hash]func(log *types.Log) (interface{}, error)

	L1StandardERC20GatewayCaller     // Read-only binding to the contract
	L1StandardERC20GatewayTransactor // Write-only binding to the contract
}

// GetAddress return L1StandardERC20Gateway's contract address.
func (o *L1StandardERC20Gateway) GetAddress() common.Address {
	return o.Address
}

// GetParsers return Parsers
func (o *L1StandardERC20Gateway) GetParsers() map[common.Hash]func(log *types.Log) (interface{}, error) {
	return o.Parsers
}

// GetABI return *big.ABI
func (o *L1StandardERC20Gateway) GetABI() *abi.ABI {
	return o.ABI
}

// L1StandardERC20GatewayCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1StandardERC20GatewayCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1StandardERC20GatewayTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1StandardERC20GatewayTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NewL1StandardERC20Gateway creates a new instance of L1StandardERC20Gateway, bound to a specific deployed contract.
func NewL1StandardERC20Gateway(address common.Address, backend bind.ContractBackend) (*L1StandardERC20Gateway, error) {
	contract, err := bindL1StandardERC20Gateway(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	sigAbi, err := L1StandardERC20GatewayMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	var parsers = map[common.Hash]func(log *types.Log) (interface{}, error){}

	parsers[sigAbi.Events["DepositERC20"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1StandardERC20GatewayDepositERC20Event)
		if err := contract.UnpackLog(event, "DepositERC20", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["FinalizeWithdrawERC20"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1StandardERC20GatewayFinalizeWithdrawERC20Event)
		if err := contract.UnpackLog(event, "FinalizeWithdrawERC20", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Initialized"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1StandardERC20GatewayInitializedEvent)
		if err := contract.UnpackLog(event, "Initialized", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["OwnershipTransferred"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1StandardERC20GatewayOwnershipTransferredEvent)
		if err := contract.UnpackLog(event, "OwnershipTransferred", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["RefundERC20"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1StandardERC20GatewayRefundERC20Event)
		if err := contract.UnpackLog(event, "RefundERC20", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	return &L1StandardERC20Gateway{ABI: sigAbi, Address: address, Parsers: parsers, L1StandardERC20GatewayCaller: L1StandardERC20GatewayCaller{contract: contract}, L1StandardERC20GatewayTransactor: L1StandardERC20GatewayTransactor{contract: contract}}, nil
}

// NewL1StandardERC20GatewayCaller creates a new read-only instance of L1StandardERC20Gateway, bound to a specific deployed contract.
func NewL1StandardERC20GatewayCaller(address common.Address, caller bind.ContractCaller) (*L1StandardERC20GatewayCaller, error) {
	contract, err := bindL1StandardERC20Gateway(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1StandardERC20GatewayCaller{contract: contract}, nil
}

// NewL1StandardERC20GatewayTransactor creates a new write-only instance of L1StandardERC20Gateway, bound to a specific deployed contract.
func NewL1StandardERC20GatewayTransactor(address common.Address, transactor bind.ContractTransactor) (*L1StandardERC20GatewayTransactor, error) {
	contract, err := bindL1StandardERC20Gateway(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1StandardERC20GatewayTransactor{contract: contract}, nil
}

// bindL1StandardERC20Gateway binds a generic wrapper to an already deployed contract.
func bindL1StandardERC20Gateway(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1StandardERC20GatewayMetaData.ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Counterpart is a free data retrieval call binding the contract method 0x797594b0.
//
// Solidity: function counterpart() view returns(address)
func (_L1StandardERC20Gateway *L1StandardERC20GatewayCaller) Counterpart(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1StandardERC20Gateway.contract.Call(opts, &out, "counterpart")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetL2ERC20Address is a free data retrieval call binding the contract method 0xc676ad29.
//
// Solidity: function getL2ERC20Address(address _l1Token) view returns(address)
func (_L1StandardERC20Gateway *L1StandardERC20GatewayCaller) GetL2ERC20Address(opts *bind.CallOpts, _l1Token common.Address) (common.Address, error) {
	var out []interface{}
	err := _L1StandardERC20Gateway.contract.Call(opts, &out, "getL2ERC20Address", _l1Token)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2TokenFactory is a free data retrieval call binding the contract method 0xeddd5e82.
//
// Solidity: function l2TokenFactory() view returns(address)
func (_L1StandardERC20Gateway *L1StandardERC20GatewayCaller) L2TokenFactory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1StandardERC20Gateway.contract.Call(opts, &out, "l2TokenFactory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2TokenImplementation is a free data retrieval call binding the contract method 0x0e28c1f2.
//
// Solidity: function l2TokenImplementation() view returns(address)
func (_L1StandardERC20Gateway *L1StandardERC20GatewayCaller) L2TokenImplementation(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1StandardERC20Gateway.contract.Call(opts, &out, "l2TokenImplementation")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1StandardERC20Gateway *L1StandardERC20GatewayCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1StandardERC20Gateway.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1StandardERC20Gateway *L1StandardERC20GatewayCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1StandardERC20Gateway.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Router is a free data retrieval call binding the contract method 0xf887ea40.
//
// Solidity: function router() view returns(address)
func (_L1StandardERC20Gateway *L1StandardERC20GatewayCaller) Router(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1StandardERC20Gateway.contract.Call(opts, &out, "router")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DepositERC20 is a paid mutator transaction binding the contract method 0x21425ee0.
//
// Solidity: function depositERC20(address _token, uint256 _amount, uint256 _gasLimit) payable returns()
func (_L1StandardERC20Gateway *L1StandardERC20GatewayTransactor) DepositERC20(opts *bind.TransactOpts, _token common.Address, _amount *big.Int, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1StandardERC20Gateway.contract.Transact(opts, "depositERC20", _token, _amount, _gasLimit)
}

// DepositERC200 is a paid mutator transaction binding the contract method 0xf219fa66.
//
// Solidity: function depositERC20(address _token, address _to, uint256 _amount, uint256 _gasLimit) payable returns()
func (_L1StandardERC20Gateway *L1StandardERC20GatewayTransactor) DepositERC200(opts *bind.TransactOpts, _token common.Address, _to common.Address, _amount *big.Int, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1StandardERC20Gateway.contract.Transact(opts, "depositERC200", _token, _to, _amount, _gasLimit)
}

// DepositERC20AndCall is a paid mutator transaction binding the contract method 0x0aea8c26.
//
// Solidity: function depositERC20AndCall(address _token, address _to, uint256 _amount, bytes _data, uint256 _gasLimit) payable returns()
func (_L1StandardERC20Gateway *L1StandardERC20GatewayTransactor) DepositERC20AndCall(opts *bind.TransactOpts, _token common.Address, _to common.Address, _amount *big.Int, _data []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1StandardERC20Gateway.contract.Transact(opts, "depositERC20AndCall", _token, _to, _amount, _data, _gasLimit)
}

// FinalizeWithdrawERC20 is a paid mutator transaction binding the contract method 0x84bd13b0.
//
// Solidity: function finalizeWithdrawERC20(address _l1Token, address _l2Token, address _from, address _to, uint256 _amount, bytes _data) payable returns()
func (_L1StandardERC20Gateway *L1StandardERC20GatewayTransactor) FinalizeWithdrawERC20(opts *bind.TransactOpts, _l1Token common.Address, _l2Token common.Address, _from common.Address, _to common.Address, _amount *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1StandardERC20Gateway.contract.Transact(opts, "finalizeWithdrawERC20", _l1Token, _l2Token, _from, _to, _amount, _data)
}

// Initialize is a paid mutator transaction binding the contract method 0x1459457a.
//
// Solidity: function initialize(address _counterpart, address _router, address _messenger, address _l2TokenImplementation, address _l2TokenFactory) returns()
func (_L1StandardERC20Gateway *L1StandardERC20GatewayTransactor) Initialize(opts *bind.TransactOpts, _counterpart common.Address, _router common.Address, _messenger common.Address, _l2TokenImplementation common.Address, _l2TokenFactory common.Address) (*types.Transaction, error) {
	return _L1StandardERC20Gateway.contract.Transact(opts, "initialize", _counterpart, _router, _messenger, _l2TokenImplementation, _l2TokenFactory)
}

// OnDropMessage is a paid mutator transaction binding the contract method 0x14298c51.
//
// Solidity: function onDropMessage(bytes _message) payable returns()
func (_L1StandardERC20Gateway *L1StandardERC20GatewayTransactor) OnDropMessage(opts *bind.TransactOpts, _message []byte) (*types.Transaction, error) {
	return _L1StandardERC20Gateway.contract.Transact(opts, "onDropMessage", _message)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1StandardERC20Gateway *L1StandardERC20GatewayTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1StandardERC20Gateway.contract.Transact(opts, "renounceOwnership")
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1StandardERC20Gateway *L1StandardERC20GatewayTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L1StandardERC20Gateway.contract.Transact(opts, "transferOwnership", newOwner)
}

// L1StandardERC20GatewayDepositERC20 represents a DepositERC20 event raised by the L1StandardERC20Gateway contract.
type L1StandardERC20GatewayDepositERC20Event struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
	raw     *types.Log // Blockchain specific contextual infos
}

// L1StandardERC20GatewayFinalizeWithdrawERC20 represents a FinalizeWithdrawERC20 event raised by the L1StandardERC20Gateway contract.
type L1StandardERC20GatewayFinalizeWithdrawERC20Event struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
	raw     *types.Log // Blockchain specific contextual infos
}

// L1StandardERC20GatewayInitialized represents a Initialized event raised by the L1StandardERC20Gateway contract.
type L1StandardERC20GatewayInitializedEvent struct {
	Version uint8
	raw     *types.Log // Blockchain specific contextual infos
}

// L1StandardERC20GatewayOwnershipTransferred represents a OwnershipTransferred event raised by the L1StandardERC20Gateway contract.
type L1StandardERC20GatewayOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	raw           *types.Log // Blockchain specific contextual infos
}

// L1StandardERC20GatewayRefundERC20 represents a RefundERC20 event raised by the L1StandardERC20Gateway contract.
type L1StandardERC20GatewayRefundERC20Event struct {
	Token     common.Address
	Recipient common.Address
	Amount    *big.Int
	raw       *types.Log // Blockchain specific contextual infos
}
