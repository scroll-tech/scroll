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

// L2GatewayRouterMetaData contains all meta data concerning the L2GatewayRouter contract.
var (
	L2GatewayRouterMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"FinalizeDepositERC20\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"FinalizeDepositETH\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Initialized\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"oldDefaultERC20Gateway\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newDefaultERC20Gateway\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"SetDefaultERC20Gateway\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"oldGateway\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newGateway\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"SetERC20Gateway\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"oldETHGateway\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newEthGateway\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"SetETHGateway\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"WithdrawERC20\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"WithdrawETH\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"ERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"defaultERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"ethGateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"finalizeDepositERC20\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"finalizeDepositETH\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"getERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l2Address\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"getL1ERC20Address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\",\"name\":\"getL2ERC20Address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_ethGateway\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_defaultERC20Gateway\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"initialize\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"renounceOwnership\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newDefaultERC20Gateway\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setDefaultERC20Gateway\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"_tokens\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"_gateways\",\"type\":\"address[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setERC20Gateway\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newEthGateway\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setETHGateway\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"transferOwnership\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"withdrawERC20\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"withdrawERC20\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"withdrawERC20AndCall\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"withdrawETH\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"withdrawETH\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"withdrawETHAndCall\"}]",
	}
	// L2GatewayRouterABI is the input ABI used to generate the binding from.
	L2GatewayRouterABI *abi.ABI

	// FinalizeDepositERC20 event
	L2GatewayRouterFinalizeDepositERC20EventSignature common.Hash

	// FinalizeDepositETH event
	L2GatewayRouterFinalizeDepositETHEventSignature common.Hash

	// Initialized event
	L2GatewayRouterInitializedEventSignature common.Hash

	// OwnershipTransferred event
	L2GatewayRouterOwnershipTransferredEventSignature common.Hash

	// SetDefaultERC20Gateway event
	L2GatewayRouterSetDefaultERC20GatewayEventSignature common.Hash

	// SetERC20Gateway event
	L2GatewayRouterSetERC20GatewayEventSignature common.Hash

	// SetETHGateway event
	L2GatewayRouterSetETHGatewayEventSignature common.Hash

	// WithdrawERC20 event
	L2GatewayRouterWithdrawERC20EventSignature common.Hash

	// WithdrawETH event
	L2GatewayRouterWithdrawETHEventSignature common.Hash
)

func init() {
	sigAbi, err := L2GatewayRouterMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	L2GatewayRouterABI = sigAbi

	// FinalizeDepositERC20 event
	L2GatewayRouterFinalizeDepositERC20EventSignature = sigAbi.Events["FinalizeDepositERC20"].ID

	// FinalizeDepositETH event
	L2GatewayRouterFinalizeDepositETHEventSignature = sigAbi.Events["FinalizeDepositETH"].ID

	// Initialized event
	L2GatewayRouterInitializedEventSignature = sigAbi.Events["Initialized"].ID

	// OwnershipTransferred event
	L2GatewayRouterOwnershipTransferredEventSignature = sigAbi.Events["OwnershipTransferred"].ID

	// SetDefaultERC20Gateway event
	L2GatewayRouterSetDefaultERC20GatewayEventSignature = sigAbi.Events["SetDefaultERC20Gateway"].ID

	// SetERC20Gateway event
	L2GatewayRouterSetERC20GatewayEventSignature = sigAbi.Events["SetERC20Gateway"].ID

	// SetETHGateway event
	L2GatewayRouterSetETHGatewayEventSignature = sigAbi.Events["SetETHGateway"].ID

	// WithdrawERC20 event
	L2GatewayRouterWithdrawERC20EventSignature = sigAbi.Events["WithdrawERC20"].ID

	// WithdrawETH event
	L2GatewayRouterWithdrawETHEventSignature = sigAbi.Events["WithdrawETH"].ID

}

// L2GatewayRouter is an auto generated Go binding around an Ethereum contract.
type L2GatewayRouter struct {
	Address common.Address // contract address
	ABI     *abi.ABI       // L2GatewayRouterABI is the input ABI used to generate the binding from.
	Parsers map[common.Hash]func(log *types.Log) (interface{}, error)

	L2GatewayRouterCaller     // Read-only binding to the contract
	L2GatewayRouterTransactor // Write-only binding to the contract
}

// GetAddress return L2GatewayRouter's contract address.
func (o *L2GatewayRouter) GetAddress() common.Address {
	return o.Address
}

// GetParsers return Parsers
func (o *L2GatewayRouter) GetParsers() map[common.Hash]func(log *types.Log) (interface{}, error) {
	return o.Parsers
}

// GetABI return *big.ABI
func (o *L2GatewayRouter) GetABI() *abi.ABI {
	return o.ABI
}

// L2GatewayRouterCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2GatewayRouterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2GatewayRouterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2GatewayRouterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NewL2GatewayRouter creates a new instance of L2GatewayRouter, bound to a specific deployed contract.
func NewL2GatewayRouter(address common.Address, backend bind.ContractBackend) (*L2GatewayRouter, error) {
	contract, err := bindL2GatewayRouter(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	sigAbi, err := L2GatewayRouterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	var parsers = map[common.Hash]func(log *types.Log) (interface{}, error){}

	parsers[sigAbi.Events["FinalizeDepositERC20"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GatewayRouterFinalizeDepositERC20Event)
		if err := contract.UnpackLog(event, "FinalizeDepositERC20", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["FinalizeDepositETH"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GatewayRouterFinalizeDepositETHEvent)
		if err := contract.UnpackLog(event, "FinalizeDepositETH", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Initialized"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GatewayRouterInitializedEvent)
		if err := contract.UnpackLog(event, "Initialized", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["OwnershipTransferred"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GatewayRouterOwnershipTransferredEvent)
		if err := contract.UnpackLog(event, "OwnershipTransferred", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["SetDefaultERC20Gateway"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GatewayRouterSetDefaultERC20GatewayEvent)
		if err := contract.UnpackLog(event, "SetDefaultERC20Gateway", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["SetERC20Gateway"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GatewayRouterSetERC20GatewayEvent)
		if err := contract.UnpackLog(event, "SetERC20Gateway", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["SetETHGateway"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GatewayRouterSetETHGatewayEvent)
		if err := contract.UnpackLog(event, "SetETHGateway", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["WithdrawERC20"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GatewayRouterWithdrawERC20Event)
		if err := contract.UnpackLog(event, "WithdrawERC20", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["WithdrawETH"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L2GatewayRouterWithdrawETHEvent)
		if err := contract.UnpackLog(event, "WithdrawETH", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	return &L2GatewayRouter{ABI: sigAbi, Address: address, Parsers: parsers, L2GatewayRouterCaller: L2GatewayRouterCaller{contract: contract}, L2GatewayRouterTransactor: L2GatewayRouterTransactor{contract: contract}}, nil
}

// NewL2GatewayRouterCaller creates a new read-only instance of L2GatewayRouter, bound to a specific deployed contract.
func NewL2GatewayRouterCaller(address common.Address, caller bind.ContractCaller) (*L2GatewayRouterCaller, error) {
	contract, err := bindL2GatewayRouter(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2GatewayRouterCaller{contract: contract}, nil
}

// NewL2GatewayRouterTransactor creates a new write-only instance of L2GatewayRouter, bound to a specific deployed contract.
func NewL2GatewayRouterTransactor(address common.Address, transactor bind.ContractTransactor) (*L2GatewayRouterTransactor, error) {
	contract, err := bindL2GatewayRouter(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2GatewayRouterTransactor{contract: contract}, nil
}

// bindL2GatewayRouter binds a generic wrapper to an already deployed contract.
func bindL2GatewayRouter(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2GatewayRouterMetaData.ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// ERC20Gateway is a free data retrieval call binding the contract method 0x705b05b8.
//
// Solidity: function ERC20Gateway(address ) view returns(address)
func (_L2GatewayRouter *L2GatewayRouterCaller) ERC20Gateway(opts *bind.CallOpts, arg0 common.Address) (common.Address, error) {
	var out []interface{}
	err := _L2GatewayRouter.contract.Call(opts, &out, "ERC20Gateway", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DefaultERC20Gateway is a free data retrieval call binding the contract method 0xce8c3e06.
//
// Solidity: function defaultERC20Gateway() view returns(address)
func (_L2GatewayRouter *L2GatewayRouterCaller) DefaultERC20Gateway(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2GatewayRouter.contract.Call(opts, &out, "defaultERC20Gateway")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// EthGateway is a free data retrieval call binding the contract method 0x8c00ce73.
//
// Solidity: function ethGateway() view returns(address)
func (_L2GatewayRouter *L2GatewayRouterCaller) EthGateway(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2GatewayRouter.contract.Call(opts, &out, "ethGateway")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetERC20Gateway is a free data retrieval call binding the contract method 0x43c66741.
//
// Solidity: function getERC20Gateway(address _token) view returns(address)
func (_L2GatewayRouter *L2GatewayRouterCaller) GetERC20Gateway(opts *bind.CallOpts, _token common.Address) (common.Address, error) {
	var out []interface{}
	err := _L2GatewayRouter.contract.Call(opts, &out, "getERC20Gateway", _token)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetL1ERC20Address is a free data retrieval call binding the contract method 0x54bbd59c.
//
// Solidity: function getL1ERC20Address(address _l2Address) view returns(address)
func (_L2GatewayRouter *L2GatewayRouterCaller) GetL1ERC20Address(opts *bind.CallOpts, _l2Address common.Address) (common.Address, error) {
	var out []interface{}
	err := _L2GatewayRouter.contract.Call(opts, &out, "getL1ERC20Address", _l2Address)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetL2ERC20Address is a free data retrieval call binding the contract method 0xc676ad29.
//
// Solidity: function getL2ERC20Address(address ) pure returns(address)
func (_L2GatewayRouter *L2GatewayRouterCaller) GetL2ERC20Address(opts *bind.CallOpts, arg0 common.Address) (common.Address, error) {
	var out []interface{}
	err := _L2GatewayRouter.contract.Call(opts, &out, "getL2ERC20Address", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2GatewayRouter *L2GatewayRouterCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2GatewayRouter.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FinalizeDepositERC20 is a paid mutator transaction binding the contract method 0x8431f5c1.
//
// Solidity: function finalizeDepositERC20(address , address , address , address , uint256 , bytes ) payable returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) FinalizeDepositERC20(opts *bind.TransactOpts, arg0 common.Address, arg1 common.Address, arg2 common.Address, arg3 common.Address, arg4 *big.Int, arg5 []byte) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "finalizeDepositERC20", arg0, arg1, arg2, arg3, arg4, arg5)
}

// FinalizeDepositETH is a paid mutator transaction binding the contract method 0x232e8748.
//
// Solidity: function finalizeDepositETH(address , address , uint256 , bytes ) payable returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) FinalizeDepositETH(opts *bind.TransactOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 []byte) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "finalizeDepositETH", arg0, arg1, arg2, arg3)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _ethGateway, address _defaultERC20Gateway) returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) Initialize(opts *bind.TransactOpts, _ethGateway common.Address, _defaultERC20Gateway common.Address) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "initialize", _ethGateway, _defaultERC20Gateway)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "renounceOwnership")
}

// SetDefaultERC20Gateway is a paid mutator transaction binding the contract method 0x5dfd5b9a.
//
// Solidity: function setDefaultERC20Gateway(address _newDefaultERC20Gateway) returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) SetDefaultERC20Gateway(opts *bind.TransactOpts, _newDefaultERC20Gateway common.Address) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "setDefaultERC20Gateway", _newDefaultERC20Gateway)
}

// SetERC20Gateway is a paid mutator transaction binding the contract method 0x635c8637.
//
// Solidity: function setERC20Gateway(address[] _tokens, address[] _gateways) returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) SetERC20Gateway(opts *bind.TransactOpts, _tokens []common.Address, _gateways []common.Address) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "setERC20Gateway", _tokens, _gateways)
}

// SetETHGateway is a paid mutator transaction binding the contract method 0x3d1d31c7.
//
// Solidity: function setETHGateway(address _newEthGateway) returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) SetETHGateway(opts *bind.TransactOpts, _newEthGateway common.Address) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "setETHGateway", _newEthGateway)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "transferOwnership", newOwner)
}

// WithdrawERC20 is a paid mutator transaction binding the contract method 0x6c07ea43.
//
// Solidity: function withdrawERC20(address _token, uint256 _amount, uint256 _gasLimit) payable returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) WithdrawERC20(opts *bind.TransactOpts, _token common.Address, _amount *big.Int, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "withdrawERC20", _token, _amount, _gasLimit)
}

// WithdrawERC200 is a paid mutator transaction binding the contract method 0xa93a4af9.
//
// Solidity: function withdrawERC20(address _token, address _to, uint256 _amount, uint256 _gasLimit) payable returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) WithdrawERC200(opts *bind.TransactOpts, _token common.Address, _to common.Address, _amount *big.Int, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "withdrawERC200", _token, _to, _amount, _gasLimit)
}

// WithdrawERC20AndCall is a paid mutator transaction binding the contract method 0x575361b6.
//
// Solidity: function withdrawERC20AndCall(address _token, address _to, uint256 _amount, bytes _data, uint256 _gasLimit) payable returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) WithdrawERC20AndCall(opts *bind.TransactOpts, _token common.Address, _to common.Address, _amount *big.Int, _data []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "withdrawERC20AndCall", _token, _to, _amount, _data, _gasLimit)
}

// WithdrawETH is a paid mutator transaction binding the contract method 0x2fcc29fa.
//
// Solidity: function withdrawETH(address _to, uint256 _amount, uint256 _gasLimit) payable returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) WithdrawETH(opts *bind.TransactOpts, _to common.Address, _amount *big.Int, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "withdrawETH", _to, _amount, _gasLimit)
}

// WithdrawETH0 is a paid mutator transaction binding the contract method 0xc7cdea37.
//
// Solidity: function withdrawETH(uint256 _amount, uint256 _gasLimit) payable returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) WithdrawETH0(opts *bind.TransactOpts, _amount *big.Int, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "withdrawETH0", _amount, _gasLimit)
}

// WithdrawETHAndCall is a paid mutator transaction binding the contract method 0x6dc24183.
//
// Solidity: function withdrawETHAndCall(address _to, uint256 _amount, bytes _data, uint256 _gasLimit) payable returns()
func (_L2GatewayRouter *L2GatewayRouterTransactor) WithdrawETHAndCall(opts *bind.TransactOpts, _to common.Address, _amount *big.Int, _data []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L2GatewayRouter.contract.Transact(opts, "withdrawETHAndCall", _to, _amount, _data, _gasLimit)
}

// L2GatewayRouterFinalizeDepositERC20 represents a FinalizeDepositERC20 event raised by the L2GatewayRouter contract.
type L2GatewayRouterFinalizeDepositERC20Event struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
}

// L2GatewayRouterFinalizeDepositETH represents a FinalizeDepositETH event raised by the L2GatewayRouter contract.
type L2GatewayRouterFinalizeDepositETHEvent struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Data   []byte
}

// L2GatewayRouterInitialized represents a Initialized event raised by the L2GatewayRouter contract.
type L2GatewayRouterInitializedEvent struct {
	Version uint8
}

// L2GatewayRouterOwnershipTransferred represents a OwnershipTransferred event raised by the L2GatewayRouter contract.
type L2GatewayRouterOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
}

// L2GatewayRouterSetDefaultERC20Gateway represents a SetDefaultERC20Gateway event raised by the L2GatewayRouter contract.
type L2GatewayRouterSetDefaultERC20GatewayEvent struct {
	OldDefaultERC20Gateway common.Address
	NewDefaultERC20Gateway common.Address
}

// L2GatewayRouterSetERC20Gateway represents a SetERC20Gateway event raised by the L2GatewayRouter contract.
type L2GatewayRouterSetERC20GatewayEvent struct {
	Token      common.Address
	OldGateway common.Address
	NewGateway common.Address
}

// L2GatewayRouterSetETHGateway represents a SetETHGateway event raised by the L2GatewayRouter contract.
type L2GatewayRouterSetETHGatewayEvent struct {
	OldETHGateway common.Address
	NewEthGateway common.Address
}

// L2GatewayRouterWithdrawERC20 represents a WithdrawERC20 event raised by the L2GatewayRouter contract.
type L2GatewayRouterWithdrawERC20Event struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
}

// L2GatewayRouterWithdrawETH represents a WithdrawETH event raised by the L2GatewayRouter contract.
type L2GatewayRouterWithdrawETHEvent struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Data   []byte
}
