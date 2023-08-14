// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package L2

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

// L2ScrollMessengerMetaData contains all meta data concerning the L2ScrollMessenger contract.
var (
	L2ScrollMessengerMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_messageQueue\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\",\"indexed\":true}],\"type\":\"event\",\"name\":\"FailedRelayedMessage\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Initialized\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Paused\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\",\"indexed\":true}],\"type\":\"event\",\"name\":\"RelayedMessage\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"SentMessage\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Unpaused\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oldFeeVault\",\"type\":\"address\",\"indexed\":false},{\"internalType\":\"address\",\"name\":\"_newFeeVault\",\"type\":\"address\",\"indexed\":false}],\"type\":\"event\",\"name\":\"UpdateFeeVault\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"oldMaxFailedExecutionTimes\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"newMaxFailedExecutionTimes\",\"type\":\"uint256\",\"indexed\":false}],\"type\":\"event\",\"name\":\"UpdateMaxFailedExecutionTimes\",\"anonymous\":false},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"counterpart\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"feeVault\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_counterpart\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"initialize\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"isL1MessageExecuted\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}]},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"isL2MessageSent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}]},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"l1MessageFailedTimes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"maxFailedExecutionTimes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"messageQueue\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"relayMessage\"},{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"renounceOwnership\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"sendMessage\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"sendMessage\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_status\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setPause\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"transferOwnership\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newFeeVault\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"updateFeeVault\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_newMaxFailedExecutionTimes\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"updateMaxFailedExecutionTimes\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"xDomainMessageSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	}
	// L2ScrollMessengerABI is the input ABI used to generate the binding from.
	L2ScrollMessengerABI *abi.ABI

	// FailedRelayedMessage event
	L2ScrollMessengerFailedRelayedMessageEventSignature common.Hash

	// Initialized event
	L2ScrollMessengerInitializedEventSignature common.Hash

	// OwnershipTransferred event
	L2ScrollMessengerOwnershipTransferredEventSignature common.Hash

	// Paused event
	L2ScrollMessengerPausedEventSignature common.Hash

	// RelayedMessage event
	L2ScrollMessengerRelayedMessageEventSignature common.Hash

	// SentMessage event
	L2ScrollMessengerSentMessageEventSignature common.Hash

	// Unpaused event
	L2ScrollMessengerUnpausedEventSignature common.Hash

	// UpdateFeeVault event
	L2ScrollMessengerUpdateFeeVaultEventSignature common.Hash

	// UpdateMaxFailedExecutionTimes event
	L2ScrollMessengerUpdateMaxFailedExecutionTimesEventSignature common.Hash
)

func init() {
	sigAbi, err := L2ScrollMessengerMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	L2ScrollMessengerABI = sigAbi

	// FailedRelayedMessage event
	L2ScrollMessengerFailedRelayedMessageEventSignature = sigAbi.Events["FailedRelayedMessage"].ID

	// Initialized event
	L2ScrollMessengerInitializedEventSignature = sigAbi.Events["Initialized"].ID

	// OwnershipTransferred event
	L2ScrollMessengerOwnershipTransferredEventSignature = sigAbi.Events["OwnershipTransferred"].ID

	// Paused event
	L2ScrollMessengerPausedEventSignature = sigAbi.Events["Paused"].ID

	// RelayedMessage event
	L2ScrollMessengerRelayedMessageEventSignature = sigAbi.Events["RelayedMessage"].ID

	// SentMessage event
	L2ScrollMessengerSentMessageEventSignature = sigAbi.Events["SentMessage"].ID

	// Unpaused event
	L2ScrollMessengerUnpausedEventSignature = sigAbi.Events["Unpaused"].ID

	// UpdateFeeVault event
	L2ScrollMessengerUpdateFeeVaultEventSignature = sigAbi.Events["UpdateFeeVault"].ID

	// UpdateMaxFailedExecutionTimes event
	L2ScrollMessengerUpdateMaxFailedExecutionTimesEventSignature = sigAbi.Events["UpdateMaxFailedExecutionTimes"].ID

}

// L2ScrollMessenger is an auto generated Go binding around an Ethereum contract.
type L2ScrollMessenger struct {
	Address common.Address // contract address
	ABI     *abi.ABI       // L2ScrollMessengerABI is the input ABI used to generate the binding from.
	Parsers map[common.Hash]func(log *types.Log) (interface{}, error)

	L2ScrollMessengerCaller     // Read-only binding to the contract
	L2ScrollMessengerTransactor // Write-only binding to the contract
}

// GetAddress return L2ScrollMessenger's contract address.
func (o *L2ScrollMessenger) GetAddress() common.Address {
	return o.Address
}

// GetParsers return Parsers
func (o *L2ScrollMessenger) GetParsers() map[common.Hash]func(log *types.Log) (interface{}, error) {
	return o.Parsers
}

// GetABI return *big.ABI
func (o *L2ScrollMessenger) GetABI() *abi.ABI {
	return o.ABI
}

// L2ScrollMessengerCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2ScrollMessengerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ScrollMessengerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2ScrollMessengerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NewL2ScrollMessenger creates a new instance of L2ScrollMessenger, bound to a specific deployed contract.
func NewL2ScrollMessenger(address common.Address, backend bind.ContractBackend) (*L2ScrollMessenger, error) {
	contract, err := bindL2ScrollMessenger(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	sigAbi, err := L2ScrollMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	var parsers = map[common.Hash]func(log *types.Log) (interface{}, error){}

	parsers[sigAbi.Events["FailedRelayedMessage"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2ScrollMessengerFailedRelayedMessageEvent)
		if err := contract.UnpackLog(event, "FailedRelayedMessage", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Initialized"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2ScrollMessengerInitializedEvent)
		if err := contract.UnpackLog(event, "Initialized", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["OwnershipTransferred"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2ScrollMessengerOwnershipTransferredEvent)
		if err := contract.UnpackLog(event, "OwnershipTransferred", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Paused"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2ScrollMessengerPausedEvent)
		if err := contract.UnpackLog(event, "Paused", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["RelayedMessage"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2ScrollMessengerRelayedMessageEvent)
		if err := contract.UnpackLog(event, "RelayedMessage", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["SentMessage"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2ScrollMessengerSentMessageEvent)
		if err := contract.UnpackLog(event, "SentMessage", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Unpaused"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2ScrollMessengerUnpausedEvent)
		if err := contract.UnpackLog(event, "Unpaused", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["UpdateFeeVault"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2ScrollMessengerUpdateFeeVaultEvent)
		if err := contract.UnpackLog(event, "UpdateFeeVault", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["UpdateMaxFailedExecutionTimes"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2ScrollMessengerUpdateMaxFailedExecutionTimesEvent)
		if err := contract.UnpackLog(event, "UpdateMaxFailedExecutionTimes", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	return &L2ScrollMessenger{ABI: sigAbi, Address: address, Parsers: parsers, L2ScrollMessengerCaller: L2ScrollMessengerCaller{contract: contract}, L2ScrollMessengerTransactor: L2ScrollMessengerTransactor{contract: contract}}, nil
}

// NewL2ScrollMessengerCaller creates a new read-only instance of L2ScrollMessenger, bound to a specific deployed contract.
func NewL2ScrollMessengerCaller(address common.Address, caller bind.ContractCaller) (*L2ScrollMessengerCaller, error) {
	contract, err := bindL2ScrollMessenger(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2ScrollMessengerCaller{contract: contract}, nil
}

// NewL2ScrollMessengerTransactor creates a new write-only instance of L2ScrollMessenger, bound to a specific deployed contract.
func NewL2ScrollMessengerTransactor(address common.Address, transactor bind.ContractTransactor) (*L2ScrollMessengerTransactor, error) {
	contract, err := bindL2ScrollMessenger(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2ScrollMessengerTransactor{contract: contract}, nil
}

// bindL2ScrollMessenger binds a generic wrapper to an already deployed contract.
func bindL2ScrollMessenger(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2ScrollMessengerMetaData.ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Counterpart is a free data retrieval call binding the contract method 0x797594b0.
//
// Solidity: function counterpart() view returns(address)
func (_L2ScrollMessenger *L2ScrollMessengerCaller) Counterpart(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ScrollMessenger.contract.Call(opts, &out, "counterpart")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FeeVault is a free data retrieval call binding the contract method 0x478222c2.
//
// Solidity: function feeVault() view returns(address)
func (_L2ScrollMessenger *L2ScrollMessengerCaller) FeeVault(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ScrollMessenger.contract.Call(opts, &out, "feeVault")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// IsL1MessageExecuted is a free data retrieval call binding the contract method 0x02345b50.
//
// Solidity: function isL1MessageExecuted(bytes32 ) view returns(bool)
func (_L2ScrollMessenger *L2ScrollMessengerCaller) IsL1MessageExecuted(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L2ScrollMessenger.contract.Call(opts, &out, "isL1MessageExecuted", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsL2MessageSent is a free data retrieval call binding the contract method 0x84a7d81f.
//
// Solidity: function isL2MessageSent(bytes32 ) view returns(bool)
func (_L2ScrollMessenger *L2ScrollMessengerCaller) IsL2MessageSent(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L2ScrollMessenger.contract.Call(opts, &out, "isL2MessageSent", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// L1MessageFailedTimes is a free data retrieval call binding the contract method 0xb0d0643a.
//
// Solidity: function l1MessageFailedTimes(bytes32 ) view returns(uint256)
func (_L2ScrollMessenger *L2ScrollMessengerCaller) L1MessageFailedTimes(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _L2ScrollMessenger.contract.Call(opts, &out, "l1MessageFailedTimes", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxFailedExecutionTimes is a free data retrieval call binding the contract method 0x6d2ab183.
//
// Solidity: function maxFailedExecutionTimes() view returns(uint256)
func (_L2ScrollMessenger *L2ScrollMessengerCaller) MaxFailedExecutionTimes(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2ScrollMessenger.contract.Call(opts, &out, "maxFailedExecutionTimes")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MessageQueue is a free data retrieval call binding the contract method 0x3b70c18a.
//
// Solidity: function messageQueue() view returns(address)
func (_L2ScrollMessenger *L2ScrollMessengerCaller) MessageQueue(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ScrollMessenger.contract.Call(opts, &out, "messageQueue")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2ScrollMessenger *L2ScrollMessengerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ScrollMessenger.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L2ScrollMessenger *L2ScrollMessengerCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _L2ScrollMessenger.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_L2ScrollMessenger *L2ScrollMessengerCaller) XDomainMessageSender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ScrollMessenger.contract.Call(opts, &out, "xDomainMessageSender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _counterpart) returns()
func (_L2ScrollMessenger *L2ScrollMessengerTransactor) Initialize(opts *bind.TransactOpts, _counterpart common.Address) (*types.Transaction, error) {
	return _L2ScrollMessenger.contract.Transact(opts, "initialize", _counterpart)
}

// RelayMessage is a paid mutator transaction binding the contract method 0x8ef1332e.
//
// Solidity: function relayMessage(address _from, address _to, uint256 _value, uint256 _nonce, bytes _message) returns()
func (_L2ScrollMessenger *L2ScrollMessengerTransactor) RelayMessage(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int, _nonce *big.Int, _message []byte) (*types.Transaction, error) {
	return _L2ScrollMessenger.contract.Transact(opts, "relayMessage", _from, _to, _value, _nonce, _message)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2ScrollMessenger *L2ScrollMessengerTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ScrollMessenger.contract.Transact(opts, "renounceOwnership")
}

// SendMessage is a paid mutator transaction binding the contract method 0x5f7b1577.
//
// Solidity: function sendMessage(address _to, uint256 _value, bytes _message, uint256 _gasLimit, address ) payable returns()
func (_L2ScrollMessenger *L2ScrollMessengerTransactor) SendMessage(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _message []byte, _gasLimit *big.Int, arg4 common.Address) (*types.Transaction, error) {
	return _L2ScrollMessenger.contract.Transact(opts, "sendMessage", _to, _value, _message, _gasLimit, arg4)
}

// SendMessage0 is a paid mutator transaction binding the contract method 0xb2267a7b.
//
// Solidity: function sendMessage(address _to, uint256 _value, bytes _message, uint256 _gasLimit) payable returns()
func (_L2ScrollMessenger *L2ScrollMessengerTransactor) SendMessage0(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _message []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L2ScrollMessenger.contract.Transact(opts, "sendMessage0", _to, _value, _message, _gasLimit)
}

// SetPause is a paid mutator transaction binding the contract method 0xbedb86fb.
//
// Solidity: function setPause(bool _status) returns()
func (_L2ScrollMessenger *L2ScrollMessengerTransactor) SetPause(opts *bind.TransactOpts, _status bool) (*types.Transaction, error) {
	return _L2ScrollMessenger.contract.Transact(opts, "setPause", _status)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2ScrollMessenger *L2ScrollMessengerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L2ScrollMessenger.contract.Transact(opts, "transferOwnership", newOwner)
}

// UpdateFeeVault is a paid mutator transaction binding the contract method 0x2a6cccb2.
//
// Solidity: function updateFeeVault(address _newFeeVault) returns()
func (_L2ScrollMessenger *L2ScrollMessengerTransactor) UpdateFeeVault(opts *bind.TransactOpts, _newFeeVault common.Address) (*types.Transaction, error) {
	return _L2ScrollMessenger.contract.Transact(opts, "updateFeeVault", _newFeeVault)
}

// UpdateMaxFailedExecutionTimes is a paid mutator transaction binding the contract method 0x7cf2e9ea.
//
// Solidity: function updateMaxFailedExecutionTimes(uint256 _newMaxFailedExecutionTimes) returns()
func (_L2ScrollMessenger *L2ScrollMessengerTransactor) UpdateMaxFailedExecutionTimes(opts *bind.TransactOpts, _newMaxFailedExecutionTimes *big.Int) (*types.Transaction, error) {
	return _L2ScrollMessenger.contract.Transact(opts, "updateMaxFailedExecutionTimes", _newMaxFailedExecutionTimes)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L2ScrollMessenger *L2ScrollMessengerTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ScrollMessenger.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// L2ScrollMessengerFailedRelayedMessage represents a FailedRelayedMessage event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerFailedRelayedMessageEvent struct {
	MessageHash [32]byte
	raw         *types.Log // Blockchain specific contextual infos
}

// L2ScrollMessengerInitialized represents a Initialized event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerInitializedEvent struct {
	Version uint8
	raw     *types.Log // Blockchain specific contextual infos
}

// L2ScrollMessengerOwnershipTransferred represents a OwnershipTransferred event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	raw           *types.Log // Blockchain specific contextual infos
}

// L2ScrollMessengerPaused represents a Paused event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerPausedEvent struct {
	Account common.Address
	raw     *types.Log // Blockchain specific contextual infos
}

// L2ScrollMessengerRelayedMessage represents a RelayedMessage event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerRelayedMessageEvent struct {
	MessageHash [32]byte
	raw         *types.Log // Blockchain specific contextual infos
}

// L2ScrollMessengerSentMessage represents a SentMessage event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerSentMessageEvent struct {
	Sender       common.Address
	Target       common.Address
	Value        *big.Int
	MessageNonce *big.Int
	GasLimit     *big.Int
	Message      []byte
	raw          *types.Log // Blockchain specific contextual infos
}

// L2ScrollMessengerUnpaused represents a Unpaused event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerUnpausedEvent struct {
	Account common.Address
	raw     *types.Log // Blockchain specific contextual infos
}

// L2ScrollMessengerUpdateFeeVault represents a UpdateFeeVault event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerUpdateFeeVaultEvent struct {
	OldFeeVault common.Address
	NewFeeVault common.Address
	raw         *types.Log // Blockchain specific contextual infos
}

// L2ScrollMessengerUpdateMaxFailedExecutionTimes represents a UpdateMaxFailedExecutionTimes event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerUpdateMaxFailedExecutionTimesEvent struct {
	OldMaxFailedExecutionTimes *big.Int
	NewMaxFailedExecutionTimes *big.Int
	raw                        *types.Log // Blockchain specific contextual infos
}
