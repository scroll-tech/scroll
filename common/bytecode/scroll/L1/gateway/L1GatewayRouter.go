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

// L1GatewayRouterMetaData contains all meta data concerning the L1GatewayRouter contract.
var (
	L1GatewayRouterMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"DepositERC20\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"DepositETH\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"FinalizeWithdrawERC20\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"FinalizeWithdrawETH\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Initialized\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false}],\"type\":\"event\",\"name\":\"RefundERC20\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false}],\"type\":\"event\",\"name\":\"RefundETH\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"oldDefaultERC20Gateway\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newDefaultERC20Gateway\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"SetDefaultERC20Gateway\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"oldGateway\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newGateway\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"SetERC20Gateway\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"oldETHGateway\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newEthGateway\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"SetETHGateway\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"ERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"defaultERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"depositERC20\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"depositERC20\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"depositERC20AndCall\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"depositETH\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"depositETH\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"depositETHAndCall\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"ethGateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"finalizeWithdrawERC20\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"finalizeWithdrawETH\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"gatewayInContext\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"getERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1Address\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"getL2ERC20Address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_ethGateway\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_defaultERC20Gateway\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"initialize\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"renounceOwnership\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"requestERC20\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newDefaultERC20Gateway\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setDefaultERC20Gateway\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"_tokens\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"_gateways\",\"type\":\"address[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setERC20Gateway\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newEthGateway\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setETHGateway\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"transferOwnership\"}]",
	}
	// L1GatewayRouterABI is the input ABI used to generate the binding from.
	L1GatewayRouterABI *abi.ABI

	// DepositERC20 event
	L1GatewayRouterDepositERC20EventSignature common.Hash

	// DepositETH event
	L1GatewayRouterDepositETHEventSignature common.Hash

	// FinalizeWithdrawERC20 event
	L1GatewayRouterFinalizeWithdrawERC20EventSignature common.Hash

	// FinalizeWithdrawETH event
	L1GatewayRouterFinalizeWithdrawETHEventSignature common.Hash

	// Initialized event
	L1GatewayRouterInitializedEventSignature common.Hash

	// OwnershipTransferred event
	L1GatewayRouterOwnershipTransferredEventSignature common.Hash

	// RefundERC20 event
	L1GatewayRouterRefundERC20EventSignature common.Hash

	// RefundETH event
	L1GatewayRouterRefundETHEventSignature common.Hash

	// SetDefaultERC20Gateway event
	L1GatewayRouterSetDefaultERC20GatewayEventSignature common.Hash

	// SetERC20Gateway event
	L1GatewayRouterSetERC20GatewayEventSignature common.Hash

	// SetETHGateway event
	L1GatewayRouterSetETHGatewayEventSignature common.Hash
)

func init() {
	sigAbi, err := L1GatewayRouterMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	L1GatewayRouterABI = sigAbi

	// DepositERC20 event
	L1GatewayRouterDepositERC20EventSignature = sigAbi.Events["DepositERC20"].ID

	// DepositETH event
	L1GatewayRouterDepositETHEventSignature = sigAbi.Events["DepositETH"].ID

	// FinalizeWithdrawERC20 event
	L1GatewayRouterFinalizeWithdrawERC20EventSignature = sigAbi.Events["FinalizeWithdrawERC20"].ID

	// FinalizeWithdrawETH event
	L1GatewayRouterFinalizeWithdrawETHEventSignature = sigAbi.Events["FinalizeWithdrawETH"].ID

	// Initialized event
	L1GatewayRouterInitializedEventSignature = sigAbi.Events["Initialized"].ID

	// OwnershipTransferred event
	L1GatewayRouterOwnershipTransferredEventSignature = sigAbi.Events["OwnershipTransferred"].ID

	// RefundERC20 event
	L1GatewayRouterRefundERC20EventSignature = sigAbi.Events["RefundERC20"].ID

	// RefundETH event
	L1GatewayRouterRefundETHEventSignature = sigAbi.Events["RefundETH"].ID

	// SetDefaultERC20Gateway event
	L1GatewayRouterSetDefaultERC20GatewayEventSignature = sigAbi.Events["SetDefaultERC20Gateway"].ID

	// SetERC20Gateway event
	L1GatewayRouterSetERC20GatewayEventSignature = sigAbi.Events["SetERC20Gateway"].ID

	// SetETHGateway event
	L1GatewayRouterSetETHGatewayEventSignature = sigAbi.Events["SetETHGateway"].ID

}

// L1GatewayRouter is an auto generated Go binding around an Ethereum contract.
type L1GatewayRouter struct {
	Address common.Address // contract address
	ABI     *abi.ABI       // L1GatewayRouterABI is the input ABI used to generate the binding from.
	Parsers map[common.Hash]func(log *types.Log) (interface{}, error)

	L1GatewayRouterCaller     // Read-only binding to the contract
	L1GatewayRouterTransactor // Write-only binding to the contract
}

// GetAddress return L1GatewayRouter's contract address.
func (o *L1GatewayRouter) GetAddress() common.Address {
	return o.Address
}

// GetParsers return Parsers
func (o *L1GatewayRouter) GetParsers() map[common.Hash]func(log *types.Log) (interface{}, error) {
	return o.Parsers
}

// GetABI return *big.ABI
func (o *L1GatewayRouter) GetABI() *abi.ABI {
	return o.ABI
}

// L1GatewayRouterCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1GatewayRouterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1GatewayRouterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1GatewayRouterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NewL1GatewayRouter creates a new instance of L1GatewayRouter, bound to a specific deployed contract.
func NewL1GatewayRouter(address common.Address, backend bind.ContractBackend) (*L1GatewayRouter, error) {
	contract, err := bindL1GatewayRouter(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	sigAbi, err := L1GatewayRouterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	var parsers = map[common.Hash]func(log *types.Log) (interface{}, error){}

	parsers[sigAbi.Events["DepositERC20"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GatewayRouterDepositERC20Event)
		if err := contract.UnpackLog(event, "DepositERC20", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["DepositETH"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GatewayRouterDepositETHEvent)
		if err := contract.UnpackLog(event, "DepositETH", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["FinalizeWithdrawERC20"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GatewayRouterFinalizeWithdrawERC20Event)
		if err := contract.UnpackLog(event, "FinalizeWithdrawERC20", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["FinalizeWithdrawETH"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GatewayRouterFinalizeWithdrawETHEvent)
		if err := contract.UnpackLog(event, "FinalizeWithdrawETH", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Initialized"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GatewayRouterInitializedEvent)
		if err := contract.UnpackLog(event, "Initialized", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["OwnershipTransferred"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GatewayRouterOwnershipTransferredEvent)
		if err := contract.UnpackLog(event, "OwnershipTransferred", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["RefundERC20"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GatewayRouterRefundERC20Event)
		if err := contract.UnpackLog(event, "RefundERC20", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["RefundETH"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GatewayRouterRefundETHEvent)
		if err := contract.UnpackLog(event, "RefundETH", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["SetDefaultERC20Gateway"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GatewayRouterSetDefaultERC20GatewayEvent)
		if err := contract.UnpackLog(event, "SetDefaultERC20Gateway", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["SetERC20Gateway"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GatewayRouterSetERC20GatewayEvent)
		if err := contract.UnpackLog(event, "SetERC20Gateway", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["SetETHGateway"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1GatewayRouterSetETHGatewayEvent)
		if err := contract.UnpackLog(event, "SetETHGateway", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	return &L1GatewayRouter{ABI: sigAbi, Address: address, Parsers: parsers, L1GatewayRouterCaller: L1GatewayRouterCaller{contract: contract}, L1GatewayRouterTransactor: L1GatewayRouterTransactor{contract: contract}}, nil
}

// NewL1GatewayRouterCaller creates a new read-only instance of L1GatewayRouter, bound to a specific deployed contract.
func NewL1GatewayRouterCaller(address common.Address, caller bind.ContractCaller) (*L1GatewayRouterCaller, error) {
	contract, err := bindL1GatewayRouter(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1GatewayRouterCaller{contract: contract}, nil
}

// NewL1GatewayRouterTransactor creates a new write-only instance of L1GatewayRouter, bound to a specific deployed contract.
func NewL1GatewayRouterTransactor(address common.Address, transactor bind.ContractTransactor) (*L1GatewayRouterTransactor, error) {
	contract, err := bindL1GatewayRouter(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1GatewayRouterTransactor{contract: contract}, nil
}

// bindL1GatewayRouter binds a generic wrapper to an already deployed contract.
func bindL1GatewayRouter(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1GatewayRouterMetaData.ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// ERC20Gateway is a free data retrieval call binding the contract method 0x705b05b8.
//
// Solidity: function ERC20Gateway(address ) view returns(address)
func (_L1GatewayRouter *L1GatewayRouterCaller) ERC20Gateway(opts *bind.CallOpts, arg0 common.Address) (common.Address, error) {
	var out []interface{}
	err := _L1GatewayRouter.contract.Call(opts, &out, "ERC20Gateway", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DefaultERC20Gateway is a free data retrieval call binding the contract method 0xce8c3e06.
//
// Solidity: function defaultERC20Gateway() view returns(address)
func (_L1GatewayRouter *L1GatewayRouterCaller) DefaultERC20Gateway(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1GatewayRouter.contract.Call(opts, &out, "defaultERC20Gateway")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// EthGateway is a free data retrieval call binding the contract method 0x8c00ce73.
//
// Solidity: function ethGateway() view returns(address)
func (_L1GatewayRouter *L1GatewayRouterCaller) EthGateway(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1GatewayRouter.contract.Call(opts, &out, "ethGateway")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GatewayInContext is a free data retrieval call binding the contract method 0x3a9a7b20.
//
// Solidity: function gatewayInContext() view returns(address)
func (_L1GatewayRouter *L1GatewayRouterCaller) GatewayInContext(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1GatewayRouter.contract.Call(opts, &out, "gatewayInContext")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetERC20Gateway is a free data retrieval call binding the contract method 0x43c66741.
//
// Solidity: function getERC20Gateway(address _token) view returns(address)
func (_L1GatewayRouter *L1GatewayRouterCaller) GetERC20Gateway(opts *bind.CallOpts, _token common.Address) (common.Address, error) {
	var out []interface{}
	err := _L1GatewayRouter.contract.Call(opts, &out, "getERC20Gateway", _token)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetL2ERC20Address is a free data retrieval call binding the contract method 0xc676ad29.
//
// Solidity: function getL2ERC20Address(address _l1Address) view returns(address)
func (_L1GatewayRouter *L1GatewayRouterCaller) GetL2ERC20Address(opts *bind.CallOpts, _l1Address common.Address) (common.Address, error) {
	var out []interface{}
	err := _L1GatewayRouter.contract.Call(opts, &out, "getL2ERC20Address", _l1Address)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1GatewayRouter *L1GatewayRouterCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1GatewayRouter.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DepositERC20 is a paid mutator transaction binding the contract method 0x21425ee0.
//
// Solidity: function depositERC20(address _token, uint256 _amount, uint256 _gasLimit) payable returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) DepositERC20(opts *bind.TransactOpts, _token common.Address, _amount *big.Int, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "depositERC20", _token, _amount, _gasLimit)
}

// DepositERC200 is a paid mutator transaction binding the contract method 0xf219fa66.
//
// Solidity: function depositERC20(address _token, address _to, uint256 _amount, uint256 _gasLimit) payable returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) DepositERC200(opts *bind.TransactOpts, _token common.Address, _to common.Address, _amount *big.Int, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "depositERC200", _token, _to, _amount, _gasLimit)
}

// DepositERC20AndCall is a paid mutator transaction binding the contract method 0x0aea8c26.
//
// Solidity: function depositERC20AndCall(address _token, address _to, uint256 _amount, bytes _data, uint256 _gasLimit) payable returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) DepositERC20AndCall(opts *bind.TransactOpts, _token common.Address, _to common.Address, _amount *big.Int, _data []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "depositERC20AndCall", _token, _to, _amount, _data, _gasLimit)
}

// DepositETH is a paid mutator transaction binding the contract method 0x9f8420b3.
//
// Solidity: function depositETH(uint256 _amount, uint256 _gasLimit) payable returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) DepositETH(opts *bind.TransactOpts, _amount *big.Int, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "depositETH", _amount, _gasLimit)
}

// DepositETH0 is a paid mutator transaction binding the contract method 0xce0b63ce.
//
// Solidity: function depositETH(address _to, uint256 _amount, uint256 _gasLimit) payable returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) DepositETH0(opts *bind.TransactOpts, _to common.Address, _amount *big.Int, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "depositETH0", _to, _amount, _gasLimit)
}

// DepositETHAndCall is a paid mutator transaction binding the contract method 0xaac476f8.
//
// Solidity: function depositETHAndCall(address _to, uint256 _amount, bytes _data, uint256 _gasLimit) payable returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) DepositETHAndCall(opts *bind.TransactOpts, _to common.Address, _amount *big.Int, _data []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "depositETHAndCall", _to, _amount, _data, _gasLimit)
}

// FinalizeWithdrawERC20 is a paid mutator transaction binding the contract method 0x84bd13b0.
//
// Solidity: function finalizeWithdrawERC20(address , address , address , address , uint256 , bytes ) payable returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) FinalizeWithdrawERC20(opts *bind.TransactOpts, arg0 common.Address, arg1 common.Address, arg2 common.Address, arg3 common.Address, arg4 *big.Int, arg5 []byte) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "finalizeWithdrawERC20", arg0, arg1, arg2, arg3, arg4, arg5)
}

// FinalizeWithdrawETH is a paid mutator transaction binding the contract method 0x8eaac8a3.
//
// Solidity: function finalizeWithdrawETH(address , address , uint256 , bytes ) payable returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) FinalizeWithdrawETH(opts *bind.TransactOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 []byte) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "finalizeWithdrawETH", arg0, arg1, arg2, arg3)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _ethGateway, address _defaultERC20Gateway) returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) Initialize(opts *bind.TransactOpts, _ethGateway common.Address, _defaultERC20Gateway common.Address) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "initialize", _ethGateway, _defaultERC20Gateway)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "renounceOwnership")
}

// RequestERC20 is a paid mutator transaction binding the contract method 0xc52a3bbc.
//
// Solidity: function requestERC20(address _sender, address _token, uint256 _amount) returns(uint256)
func (_L1GatewayRouter *L1GatewayRouterTransactor) RequestERC20(opts *bind.TransactOpts, _sender common.Address, _token common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "requestERC20", _sender, _token, _amount)
}

// SetDefaultERC20Gateway is a paid mutator transaction binding the contract method 0x5dfd5b9a.
//
// Solidity: function setDefaultERC20Gateway(address _newDefaultERC20Gateway) returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) SetDefaultERC20Gateway(opts *bind.TransactOpts, _newDefaultERC20Gateway common.Address) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "setDefaultERC20Gateway", _newDefaultERC20Gateway)
}

// SetERC20Gateway is a paid mutator transaction binding the contract method 0x635c8637.
//
// Solidity: function setERC20Gateway(address[] _tokens, address[] _gateways) returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) SetERC20Gateway(opts *bind.TransactOpts, _tokens []common.Address, _gateways []common.Address) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "setERC20Gateway", _tokens, _gateways)
}

// SetETHGateway is a paid mutator transaction binding the contract method 0x3d1d31c7.
//
// Solidity: function setETHGateway(address _newEthGateway) returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) SetETHGateway(opts *bind.TransactOpts, _newEthGateway common.Address) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "setETHGateway", _newEthGateway)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1GatewayRouter *L1GatewayRouterTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L1GatewayRouter.contract.Transact(opts, "transferOwnership", newOwner)
}

// L1GatewayRouterDepositERC20 represents a DepositERC20 event raised by the L1GatewayRouter contract.
type L1GatewayRouterDepositERC20Event struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
}

// L1GatewayRouterDepositETH represents a DepositETH event raised by the L1GatewayRouter contract.
type L1GatewayRouterDepositETHEvent struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Data   []byte
}

// L1GatewayRouterFinalizeWithdrawERC20 represents a FinalizeWithdrawERC20 event raised by the L1GatewayRouter contract.
type L1GatewayRouterFinalizeWithdrawERC20Event struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
}

// L1GatewayRouterFinalizeWithdrawETH represents a FinalizeWithdrawETH event raised by the L1GatewayRouter contract.
type L1GatewayRouterFinalizeWithdrawETHEvent struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Data   []byte
}

// L1GatewayRouterInitialized represents a Initialized event raised by the L1GatewayRouter contract.
type L1GatewayRouterInitializedEvent struct {
	Version uint8
}

// L1GatewayRouterOwnershipTransferred represents a OwnershipTransferred event raised by the L1GatewayRouter contract.
type L1GatewayRouterOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
}

// L1GatewayRouterRefundERC20 represents a RefundERC20 event raised by the L1GatewayRouter contract.
type L1GatewayRouterRefundERC20Event struct {
	Token     common.Address
	Recipient common.Address
	Amount    *big.Int
}

// L1GatewayRouterRefundETH represents a RefundETH event raised by the L1GatewayRouter contract.
type L1GatewayRouterRefundETHEvent struct {
	Recipient common.Address
	Amount    *big.Int
}

// L1GatewayRouterSetDefaultERC20Gateway represents a SetDefaultERC20Gateway event raised by the L1GatewayRouter contract.
type L1GatewayRouterSetDefaultERC20GatewayEvent struct {
	OldDefaultERC20Gateway common.Address
	NewDefaultERC20Gateway common.Address
}

// L1GatewayRouterSetERC20Gateway represents a SetERC20Gateway event raised by the L1GatewayRouter contract.
type L1GatewayRouterSetERC20GatewayEvent struct {
	Token      common.Address
	OldGateway common.Address
	NewGateway common.Address
}

// L1GatewayRouterSetETHGateway represents a SetETHGateway event raised by the L1GatewayRouter contract.
type L1GatewayRouterSetETHGatewayEvent struct {
	OldETHGateway common.Address
	NewEthGateway common.Address
}
