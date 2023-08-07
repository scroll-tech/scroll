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

// L1GatewayRouterMetaData contains all meta data concerning the L1GatewayRouter contract.
var (
	L1GatewayRouterMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"DepositERC20\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"DepositETH\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l1Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"l2Token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"FinalizeWithdrawERC20\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"FinalizeWithdrawETH\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"RefundERC20\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"RefundETH\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldDefaultERC20Gateway\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newDefaultERC20Gateway\",\"type\":\"address\"}],\"name\":\"SetDefaultERC20Gateway\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldGateway\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newGateway\",\"type\":\"address\"}],\"name\":\"SetERC20Gateway\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldETHGateway\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newEthGateway\",\"type\":\"address\"}],\"name\":\"SetETHGateway\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"ERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"defaultERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"depositERC20\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"depositERC20\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"depositERC20AndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"depositETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"depositETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"depositETHAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ethGateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"finalizeWithdrawERC20\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"finalizeWithdrawETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gatewayInContext\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"}],\"name\":\"getERC20Gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1Address\",\"type\":\"address\"}],\"name\":\"getL2ERC20Address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_ethGateway\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_defaultERC20Gateway\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"requestERC20\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newDefaultERC20Gateway\",\"type\":\"address\"}],\"name\":\"setDefaultERC20Gateway\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"_tokens\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"_gateways\",\"type\":\"address[]\"}],\"name\":\"setERC20Gateway\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newEthGateway\",\"type\":\"address\"}],\"name\":\"setETHGateway\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
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

// DepositERC20 represents a DepositERC20 event raised by the L1GatewayRouter contract.
type L1GatewayRouterDepositERC20Event struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
}

// DepositETH represents a DepositETH event raised by the L1GatewayRouter contract.
type L1GatewayRouterDepositETHEvent struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Data   []byte
}

// FinalizeWithdrawERC20 represents a FinalizeWithdrawERC20 event raised by the L1GatewayRouter contract.
type L1GatewayRouterFinalizeWithdrawERC20Event struct {
	L1Token common.Address
	L2Token common.Address
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
}

// FinalizeWithdrawETH represents a FinalizeWithdrawETH event raised by the L1GatewayRouter contract.
type L1GatewayRouterFinalizeWithdrawETHEvent struct {
	From   common.Address
	To     common.Address
	Amount *big.Int
	Data   []byte
}

// Initialized represents a Initialized event raised by the L1GatewayRouter contract.
type L1GatewayRouterInitializedEvent struct {
	Version uint8
}

// OwnershipTransferred represents a OwnershipTransferred event raised by the L1GatewayRouter contract.
type L1GatewayRouterOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
}

// RefundERC20 represents a RefundERC20 event raised by the L1GatewayRouter contract.
type L1GatewayRouterRefundERC20Event struct {
	Token     common.Address
	Recipient common.Address
	Amount    *big.Int
}

// RefundETH represents a RefundETH event raised by the L1GatewayRouter contract.
type L1GatewayRouterRefundETHEvent struct {
	Recipient common.Address
	Amount    *big.Int
}

// SetDefaultERC20Gateway represents a SetDefaultERC20Gateway event raised by the L1GatewayRouter contract.
type L1GatewayRouterSetDefaultERC20GatewayEvent struct {
	OldDefaultERC20Gateway common.Address
	NewDefaultERC20Gateway common.Address
}

// SetERC20Gateway represents a SetERC20Gateway event raised by the L1GatewayRouter contract.
type L1GatewayRouterSetERC20GatewayEvent struct {
	Token      common.Address
	OldGateway common.Address
	NewGateway common.Address
}

// SetETHGateway represents a SetETHGateway event raised by the L1GatewayRouter contract.
type L1GatewayRouterSetETHGatewayEvent struct {
	OldETHGateway common.Address
	NewEthGateway common.Address
}
