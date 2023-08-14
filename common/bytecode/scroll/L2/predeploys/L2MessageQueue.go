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

// L2MessageQueueMetaData contains all meta data concerning the L2MessageQueue contract.
var (
	L2MessageQueueMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\",\"indexed\":false}],\"type\":\"event\",\"name\":\"AppendMessage\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oldOwner\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_messageHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"appendMessage\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}]},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"branches\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_messenger\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"initialize\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"messageRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"nextMessageIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"renounceOwnership\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"transferOwnership\"}]",
	}
	// L2MessageQueueABI is the input ABI used to generate the binding from.
	L2MessageQueueABI *abi.ABI

	// AppendMessage event
	L2MessageQueueAppendMessageEventSignature common.Hash

	// OwnershipTransferred event
	L2MessageQueueOwnershipTransferredEventSignature common.Hash
)

func init() {
	sigAbi, err := L2MessageQueueMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	L2MessageQueueABI = sigAbi

	// AppendMessage event
	L2MessageQueueAppendMessageEventSignature = sigAbi.Events["AppendMessage"].ID

	// OwnershipTransferred event
	L2MessageQueueOwnershipTransferredEventSignature = sigAbi.Events["OwnershipTransferred"].ID

}

// L2MessageQueue is an auto generated Go binding around an Ethereum contract.
type L2MessageQueue struct {
	Address common.Address // contract address
	ABI     *abi.ABI       // L2MessageQueueABI is the input ABI used to generate the binding from.
	Parsers map[common.Hash]func(log *types.Log) (interface{}, error)

	L2MessageQueueCaller     // Read-only binding to the contract
	L2MessageQueueTransactor // Write-only binding to the contract
}

// GetAddress return L2MessageQueue's contract address.
func (o *L2MessageQueue) GetAddress() common.Address {
	return o.Address
}

// GetParsers return Parsers
func (o *L2MessageQueue) GetParsers() map[common.Hash]func(log *types.Log) (interface{}, error) {
	return o.Parsers
}

// GetABI return *big.ABI
func (o *L2MessageQueue) GetABI() *abi.ABI {
	return o.ABI
}

// L2MessageQueueCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2MessageQueueCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2MessageQueueTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2MessageQueueTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NewL2MessageQueue creates a new instance of L2MessageQueue, bound to a specific deployed contract.
func NewL2MessageQueue(address common.Address, backend bind.ContractBackend) (*L2MessageQueue, error) {
	contract, err := bindL2MessageQueue(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	sigAbi, err := L2MessageQueueMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	var parsers = map[common.Hash]func(log *types.Log) (interface{}, error){}

	parsers[sigAbi.Events["AppendMessage"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2MessageQueueAppendMessageEvent)
		if err := contract.UnpackLog(event, "AppendMessage", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["OwnershipTransferred"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2MessageQueueOwnershipTransferredEvent)
		if err := contract.UnpackLog(event, "OwnershipTransferred", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	return &L2MessageQueue{ABI: sigAbi, Address: address, Parsers: parsers, L2MessageQueueCaller: L2MessageQueueCaller{contract: contract}, L2MessageQueueTransactor: L2MessageQueueTransactor{contract: contract}}, nil
}

// NewL2MessageQueueCaller creates a new read-only instance of L2MessageQueue, bound to a specific deployed contract.
func NewL2MessageQueueCaller(address common.Address, caller bind.ContractCaller) (*L2MessageQueueCaller, error) {
	contract, err := bindL2MessageQueue(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2MessageQueueCaller{contract: contract}, nil
}

// NewL2MessageQueueTransactor creates a new write-only instance of L2MessageQueue, bound to a specific deployed contract.
func NewL2MessageQueueTransactor(address common.Address, transactor bind.ContractTransactor) (*L2MessageQueueTransactor, error) {
	contract, err := bindL2MessageQueue(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2MessageQueueTransactor{contract: contract}, nil
}

// bindL2MessageQueue binds a generic wrapper to an already deployed contract.
func bindL2MessageQueue(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2MessageQueueMetaData.ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Branches is a free data retrieval call binding the contract method 0x83cc7660.
//
// Solidity: function branches(uint256 ) view returns(bytes32)
func (_L2MessageQueue *L2MessageQueueCaller) Branches(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _L2MessageQueue.contract.Call(opts, &out, "branches", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// MessageRoot is a free data retrieval call binding the contract method 0xd4b9f4fa.
//
// Solidity: function messageRoot() view returns(bytes32)
func (_L2MessageQueue *L2MessageQueueCaller) MessageRoot(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L2MessageQueue.contract.Call(opts, &out, "messageRoot")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2MessageQueue *L2MessageQueueCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2MessageQueue.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NextMessageIndex is a free data retrieval call binding the contract method 0x26aad7b7.
//
// Solidity: function nextMessageIndex() view returns(uint256)
func (_L2MessageQueue *L2MessageQueueCaller) NextMessageIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2MessageQueue.contract.Call(opts, &out, "nextMessageIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2MessageQueue *L2MessageQueueCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2MessageQueue.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AppendMessage is a paid mutator transaction binding the contract method 0x600a2e77.
//
// Solidity: function appendMessage(bytes32 _messageHash) returns(bytes32)
func (_L2MessageQueue *L2MessageQueueTransactor) AppendMessage(opts *bind.TransactOpts, _messageHash [32]byte) (*types.Transaction, error) {
	return _L2MessageQueue.contract.Transact(opts, "appendMessage", _messageHash)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _messenger) returns()
func (_L2MessageQueue *L2MessageQueueTransactor) Initialize(opts *bind.TransactOpts, _messenger common.Address) (*types.Transaction, error) {
	return _L2MessageQueue.contract.Transact(opts, "initialize", _messenger)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2MessageQueue *L2MessageQueueTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2MessageQueue.contract.Transact(opts, "renounceOwnership")
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_L2MessageQueue *L2MessageQueueTransactor) TransferOwnership(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _L2MessageQueue.contract.Transact(opts, "transferOwnership", _newOwner)
}

// L2MessageQueueAppendMessage represents a AppendMessage event raised by the L2MessageQueue contract.
type L2MessageQueueAppendMessageEvent struct {
	Index       *big.Int
	MessageHash [32]byte
}

// L2MessageQueueOwnershipTransferred represents a OwnershipTransferred event raised by the L2MessageQueue contract.
type L2MessageQueueOwnershipTransferredEvent struct {
	OldOwner common.Address
	NewOwner common.Address
}
