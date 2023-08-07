// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package L2

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

// L2ScrollMessengerMetaData contains all meta data concerning the L2ScrollMessenger contract.
var (
	L2ScrollMessengerMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_messageQueue\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"FailedRelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"RelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"SentMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_oldFeeVault\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_newFeeVault\",\"type\":\"address\"}],\"name\":\"UpdateFeeVault\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"maxFailedExecutionTimes\",\"type\":\"uint256\"}],\"name\":\"UpdateMaxFailedExecutionTimes\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"counterpart\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeVault\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_counterpart\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_feeVault\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"isL1MessageExecuted\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"isL2MessageSent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"l1MessageFailedTimes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxFailedExecutionTimes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messageQueue\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"name\":\"relayMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_status\",\"type\":\"bool\"}],\"name\":\"setPause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newFeeVault\",\"type\":\"address\"}],\"name\":\"updateFeeVault\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_maxFailedExecutionTimes\",\"type\":\"uint256\"}],\"name\":\"updateMaxFailedExecutionTimes\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"xDomainMessageSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
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

// FailedRelayedMessage represents a FailedRelayedMessage event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerFailedRelayedMessageEvent struct {
	MessageHash [32]byte
}

// Initialized represents a Initialized event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerInitializedEvent struct {
	Version uint8
}

// OwnershipTransferred represents a OwnershipTransferred event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
}

// Paused represents a Paused event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerPausedEvent struct {
	Account common.Address
}

// RelayedMessage represents a RelayedMessage event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerRelayedMessageEvent struct {
	MessageHash [32]byte
}

// SentMessage represents a SentMessage event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerSentMessageEvent struct {
	Sender       common.Address
	Target       common.Address
	Value        *big.Int
	MessageNonce *big.Int
	GasLimit     *big.Int
	Message      []byte
}

// Unpaused represents a Unpaused event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerUnpausedEvent struct {
	Account common.Address
}

// UpdateFeeVault represents a UpdateFeeVault event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerUpdateFeeVaultEvent struct {
	OldFeeVault common.Address
	NewFeeVault common.Address
}

// UpdateMaxFailedExecutionTimes represents a UpdateMaxFailedExecutionTimes event raised by the L2ScrollMessenger contract.
type L2ScrollMessengerUpdateMaxFailedExecutionTimesEvent struct {
	MaxFailedExecutionTimes *big.Int
}
