// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package greeter

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

// GreeterMetaData contains all meta data concerning the Greeter contract.
var GreeterMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"num\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"retrieve\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"retrieve_failing\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"num\",\"type\":\"uint256\"}],\"name\":\"set_value\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"num\",\"type\":\"uint256\"}],\"name\":\"set_value_failing\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Sigs: map[string]string{
		"2e64cec1": "retrieve()",
		"f3417673": "retrieve_failing()",
		"b0f2b72a": "set_value(uint256)",
		"21848c46": "set_value_failing(uint256)",
	},
	Bin: "0x608060405234801561001057600080fd5b5060405161014238038061014283398101604081905261002f91610037565b600055610050565b60006020828403121561004957600080fd5b5051919050565b60e48061005e6000396000f3fe6080604052348015600f57600080fd5b506004361060465760003560e01c806321848c4614604b5780632e64cec114605c578063b0f2b72a146072578063f3417673146082575b600080fd5b605a60563660046096565b6088565b005b6000545b60405190815260200160405180910390f35b605a607d3660046096565b600055565b60606090565b600081815580fd5b60008080fd5b60006020828403121560a757600080fd5b503591905056fea26469706673582212204921de3d5e4e7973f5637bdad02a50aa0fabff6466686fd0fa8fe9561322333364736f6c634300080c0033",
}

// GreeterABI is the input ABI used to generate the binding from.
// Deprecated: Use GreeterMetaData.ABI instead.
var GreeterABI = GreeterMetaData.ABI

// Deprecated: Use GreeterMetaData.Sigs instead.
// GreeterFuncSigs maps the 4-byte function signature to its string representation.
var GreeterFuncSigs = GreeterMetaData.Sigs

// GreeterBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use GreeterMetaData.Bin instead.
var GreeterBin = GreeterMetaData.Bin

// DeployGreeter deploys a new Ethereum contract, binding an instance of Greeter to it.
func DeployGreeter(auth *bind.TransactOpts, backend bind.ContractBackend, num *big.Int) (common.Address, *types.Transaction, *Greeter, error) {
	parsed, err := GreeterMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(GreeterBin), backend, num)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Greeter{GreeterCaller: GreeterCaller{contract: contract}, GreeterTransactor: GreeterTransactor{contract: contract}, GreeterFilterer: GreeterFilterer{contract: contract}}, nil
}

// Greeter is an auto generated Go binding around an Ethereum contract.
type Greeter struct {
	GreeterCaller     // Read-only binding to the contract
	GreeterTransactor // Write-only binding to the contract
	GreeterFilterer   // Log filterer for contract events
}

// GreeterCaller is an auto generated read-only Go binding around an Ethereum contract.
type GreeterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GreeterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type GreeterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GreeterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type GreeterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GreeterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type GreeterSession struct {
	Contract     *Greeter          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// GreeterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type GreeterCallerSession struct {
	Contract *GreeterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// GreeterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type GreeterTransactorSession struct {
	Contract     *GreeterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// GreeterRaw is an auto generated low-level Go binding around an Ethereum contract.
type GreeterRaw struct {
	Contract *Greeter // Generic contract binding to access the raw methods on
}

// GreeterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type GreeterCallerRaw struct {
	Contract *GreeterCaller // Generic read-only contract binding to access the raw methods on
}

// GreeterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type GreeterTransactorRaw struct {
	Contract *GreeterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewGreeter creates a new instance of Greeter, bound to a specific deployed contract.
func NewGreeter(address common.Address, backend bind.ContractBackend) (*Greeter, error) {
	contract, err := bindGreeter(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Greeter{GreeterCaller: GreeterCaller{contract: contract}, GreeterTransactor: GreeterTransactor{contract: contract}, GreeterFilterer: GreeterFilterer{contract: contract}}, nil
}

// NewGreeterCaller creates a new read-only instance of Greeter, bound to a specific deployed contract.
func NewGreeterCaller(address common.Address, caller bind.ContractCaller) (*GreeterCaller, error) {
	contract, err := bindGreeter(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &GreeterCaller{contract: contract}, nil
}

// NewGreeterTransactor creates a new write-only instance of Greeter, bound to a specific deployed contract.
func NewGreeterTransactor(address common.Address, transactor bind.ContractTransactor) (*GreeterTransactor, error) {
	contract, err := bindGreeter(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &GreeterTransactor{contract: contract}, nil
}

// NewGreeterFilterer creates a new log filterer instance of Greeter, bound to a specific deployed contract.
func NewGreeterFilterer(address common.Address, filterer bind.ContractFilterer) (*GreeterFilterer, error) {
	contract, err := bindGreeter(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &GreeterFilterer{contract: contract}, nil
}

// bindGreeter binds a generic wrapper to an already deployed contract.
func bindGreeter(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(GreeterABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Greeter *GreeterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Greeter.Contract.GreeterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Greeter *GreeterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Greeter.Contract.GreeterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Greeter *GreeterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Greeter.Contract.GreeterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Greeter *GreeterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Greeter.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Greeter *GreeterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Greeter.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Greeter *GreeterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Greeter.Contract.contract.Transact(opts, method, params...)
}

// Retrieve is a free data retrieval call binding the contract method 0x2e64cec1.
//
// Solidity: function retrieve() view returns(uint256)
func (_Greeter *GreeterCaller) Retrieve(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Greeter.contract.Call(opts, &out, "retrieve")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Retrieve is a free data retrieval call binding the contract method 0x2e64cec1.
//
// Solidity: function retrieve() view returns(uint256)
func (_Greeter *GreeterSession) Retrieve() (*big.Int, error) {
	return _Greeter.Contract.Retrieve(&_Greeter.CallOpts)
}

// Retrieve is a free data retrieval call binding the contract method 0x2e64cec1.
//
// Solidity: function retrieve() view returns(uint256)
func (_Greeter *GreeterCallerSession) Retrieve() (*big.Int, error) {
	return _Greeter.Contract.Retrieve(&_Greeter.CallOpts)
}

// RetrieveFailing is a free data retrieval call binding the contract method 0xf3417673.
//
// Solidity: function retrieve_failing() view returns(uint256)
func (_Greeter *GreeterCaller) RetrieveFailing(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Greeter.contract.Call(opts, &out, "retrieve_failing")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RetrieveFailing is a free data retrieval call binding the contract method 0xf3417673.
//
// Solidity: function retrieve_failing() view returns(uint256)
func (_Greeter *GreeterSession) RetrieveFailing() (*big.Int, error) {
	return _Greeter.Contract.RetrieveFailing(&_Greeter.CallOpts)
}

// RetrieveFailing is a free data retrieval call binding the contract method 0xf3417673.
//
// Solidity: function retrieve_failing() view returns(uint256)
func (_Greeter *GreeterCallerSession) RetrieveFailing() (*big.Int, error) {
	return _Greeter.Contract.RetrieveFailing(&_Greeter.CallOpts)
}

// SetValue is a paid mutator transaction binding the contract method 0xb0f2b72a.
//
// Solidity: function set_value(uint256 num) returns()
func (_Greeter *GreeterTransactor) SetValue(opts *bind.TransactOpts, num *big.Int) (*types.Transaction, error) {
	return _Greeter.contract.Transact(opts, "set_value", num)
}

// SetValue is a paid mutator transaction binding the contract method 0xb0f2b72a.
//
// Solidity: function set_value(uint256 num) returns()
func (_Greeter *GreeterSession) SetValue(num *big.Int) (*types.Transaction, error) {
	return _Greeter.Contract.SetValue(&_Greeter.TransactOpts, num)
}

// SetValue is a paid mutator transaction binding the contract method 0xb0f2b72a.
//
// Solidity: function set_value(uint256 num) returns()
func (_Greeter *GreeterTransactorSession) SetValue(num *big.Int) (*types.Transaction, error) {
	return _Greeter.Contract.SetValue(&_Greeter.TransactOpts, num)
}

// SetValueFailing is a paid mutator transaction binding the contract method 0x21848c46.
//
// Solidity: function set_value_failing(uint256 num) returns()
func (_Greeter *GreeterTransactor) SetValueFailing(opts *bind.TransactOpts, num *big.Int) (*types.Transaction, error) {
	return _Greeter.contract.Transact(opts, "set_value_failing", num)
}

// SetValueFailing is a paid mutator transaction binding the contract method 0x21848c46.
//
// Solidity: function set_value_failing(uint256 num) returns()
func (_Greeter *GreeterSession) SetValueFailing(num *big.Int) (*types.Transaction, error) {
	return _Greeter.Contract.SetValueFailing(&_Greeter.TransactOpts, num)
}

// SetValueFailing is a paid mutator transaction binding the contract method 0x21848c46.
//
// Solidity: function set_value_failing(uint256 num) returns()
func (_Greeter *GreeterTransactorSession) SetValueFailing(num *big.Int) (*types.Transaction, error) {
	return _Greeter.Contract.SetValueFailing(&_Greeter.TransactOpts, num)
}
