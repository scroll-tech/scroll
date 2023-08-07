// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package gateway

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

// L2GatewayRouterMetaData contains all meta data concerning the L2GatewayRouter contract.
var (
	L2GatewayRouterMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"FinalizeDepositERC20\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"FinalizeDepositETH\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldDefaultERC20Gateway\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newDefaultERC20Gateway\",\"type\":\"address\"}],\"name\":\"SetDefaultERC20Gateway\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldGateway\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newGateway\",\"type\":\"address\"}],\"name\":\"SetERC20Gateway\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldETHGateway\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newEthGateway\",\"type\":\"address\"}],\"name\":\"SetETHGateway\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"WithdrawERC20\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"WithdrawETH\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"ERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"defaultERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ethGateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"finalizeDepositERC20\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"finalizeDepositETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"getERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l2Address\",\"type\":\"address\"}],\"name\":\"getL1ERC20Address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"getL2ERC20Address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_ethGateway\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_defaultERC20Gateway\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newDefaultERC20Gateway\",\"type\":\"address\"}],\"name\":\"setDefaultERC20Gateway\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"_tokens\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"_gateways\",\"type\":\"address[]\"}],\"name\":\"setERC20Gateway\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newEthGateway\",\"type\":\"address\"}],\"name\":\"setETHGateway\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"withdrawERC20\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"withdrawERC20\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"withdrawERC20AndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"withdrawETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"withdrawETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"withdrawETHAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
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

// FinalizeDepositERC20 represents a FinalizeDepositERC20 event raised by the L2GatewayRouter contract.
type L2GatewayRouterFinalizeDepositERC20Event struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
}

// FinalizeDepositETH represents a FinalizeDepositETH event raised by the L2GatewayRouter contract.
type L2GatewayRouterFinalizeDepositETHEvent struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Data   []byte
}

// Initialized represents a Initialized event raised by the L2GatewayRouter contract.
type L2GatewayRouterInitializedEvent struct {
	Version uint8
}

// OwnershipTransferred represents a OwnershipTransferred event raised by the L2GatewayRouter contract.
type L2GatewayRouterOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
}

// SetDefaultERC20Gateway represents a SetDefaultERC20Gateway event raised by the L2GatewayRouter contract.
type L2GatewayRouterSetDefaultERC20GatewayEvent struct {
	OldDefaultERC20Gateway common.Address
	NewDefaultERC20Gateway common.Address
}

// SetERC20Gateway represents a SetERC20Gateway event raised by the L2GatewayRouter contract.
type L2GatewayRouterSetERC20GatewayEvent struct {
	Token      common.Address
	OldGateway common.Address
	NewGateway common.Address
}

// SetETHGateway represents a SetETHGateway event raised by the L2GatewayRouter contract.
type L2GatewayRouterSetETHGatewayEvent struct {
	OldETHGateway common.Address
	NewEthGateway common.Address
}

// WithdrawERC20 represents a WithdrawERC20 event raised by the L2GatewayRouter contract.
type L2GatewayRouterWithdrawERC20Event struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
}

// WithdrawETH represents a WithdrawETH event raised by the L2GatewayRouter contract.
type L2GatewayRouterWithdrawETHEvent struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Data   []byte
}
