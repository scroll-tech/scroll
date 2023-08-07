// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package L1

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

// IL1ScrollMessengerL2MessageProof is an auto generated low-level Go binding around an user-defined struct.
type IL1ScrollMessengerL2MessageProof struct {
	BatchIndex  *big.Int
	MerkleProof []byte
}

// L1ScrollMessengerMetaData contains all meta data concerning the L1ScrollMessenger contract.
var (
	L1ScrollMessengerMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"FailedRelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"RelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"SentMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_oldFeeVault\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_newFeeVault\",\"type\":\"address\"}],\"name\":\"UpdateFeeVault\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"maxReplayTimes\",\"type\":\"uint256\"}],\"name\":\"UpdateMaxReplayTimes\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"counterpart\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_messageNonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"name\":\"dropMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"feeVault\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_counterpart\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_feeVault\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_rollup\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_messageQueue\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"isL1MessageDropped\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"isL1MessageSent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"isL2MessageExecuted\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxReplayTimes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messageQueue\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"prevReplayIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"merkleProof\",\"type\":\"bytes\"}],\"internalType\":\"structIL1ScrollMessenger.L2MessageProof\",\"name\":\"_proof\",\"type\":\"tuple\"}],\"name\":\"relayMessageWithProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_messageNonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint32\",\"name\":\"_newGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"address\",\"name\":\"_refundAddress\",\"type\":\"address\"}],\"name\":\"replayMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"replayStates\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"times\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"lastIndex\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_refundAddress\",\"type\":\"address\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_status\",\"type\":\"bool\"}],\"name\":\"setPause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newFeeVault\",\"type\":\"address\"}],\"name\":\"updateFeeVault\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_maxReplayTimes\",\"type\":\"uint256\"}],\"name\":\"updateMaxReplayTimes\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"xDomainMessageSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	}
	// L1ScrollMessengerABI is the input ABI used to generate the binding from.
	L1ScrollMessengerABI *abi.ABI

	// FailedRelayedMessage event
	L1ScrollMessengerFailedRelayedMessageEventSignature common.Hash

	// Initialized event
	L1ScrollMessengerInitializedEventSignature common.Hash

	// OwnershipTransferred event
	L1ScrollMessengerOwnershipTransferredEventSignature common.Hash

	// Paused event
	L1ScrollMessengerPausedEventSignature common.Hash

	// RelayedMessage event
	L1ScrollMessengerRelayedMessageEventSignature common.Hash

	// SentMessage event
	L1ScrollMessengerSentMessageEventSignature common.Hash

	// Unpaused event
	L1ScrollMessengerUnpausedEventSignature common.Hash

	// UpdateFeeVault event
	L1ScrollMessengerUpdateFeeVaultEventSignature common.Hash

	// UpdateMaxReplayTimes event
	L1ScrollMessengerUpdateMaxReplayTimesEventSignature common.Hash
)

func init() {
	sigAbi, err := L1ScrollMessengerMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	L1ScrollMessengerABI = sigAbi

	// FailedRelayedMessage event
	L1ScrollMessengerFailedRelayedMessageEventSignature = sigAbi.Events["FailedRelayedMessage"].ID

	// Initialized event
	L1ScrollMessengerInitializedEventSignature = sigAbi.Events["Initialized"].ID

	// OwnershipTransferred event
	L1ScrollMessengerOwnershipTransferredEventSignature = sigAbi.Events["OwnershipTransferred"].ID

	// Paused event
	L1ScrollMessengerPausedEventSignature = sigAbi.Events["Paused"].ID

	// RelayedMessage event
	L1ScrollMessengerRelayedMessageEventSignature = sigAbi.Events["RelayedMessage"].ID

	// SentMessage event
	L1ScrollMessengerSentMessageEventSignature = sigAbi.Events["SentMessage"].ID

	// Unpaused event
	L1ScrollMessengerUnpausedEventSignature = sigAbi.Events["Unpaused"].ID

	// UpdateFeeVault event
	L1ScrollMessengerUpdateFeeVaultEventSignature = sigAbi.Events["UpdateFeeVault"].ID

	// UpdateMaxReplayTimes event
	L1ScrollMessengerUpdateMaxReplayTimesEventSignature = sigAbi.Events["UpdateMaxReplayTimes"].ID

}

// FailedRelayedMessage represents a FailedRelayedMessage event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerFailedRelayedMessageEvent struct {
	MessageHash [32]byte
}

// Initialized represents a Initialized event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerInitializedEvent struct {
	Version uint8
}

// OwnershipTransferred represents a OwnershipTransferred event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
}

// Paused represents a Paused event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerPausedEvent struct {
	Account common.Address
}

// RelayedMessage represents a RelayedMessage event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerRelayedMessageEvent struct {
	MessageHash [32]byte
}

// SentMessage represents a SentMessage event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerSentMessageEvent struct {
	Sender       common.Address
	Target       common.Address
	Value        *big.Int
	MessageNonce *big.Int
	GasLimit     *big.Int
	Message      []byte
}

// Unpaused represents a Unpaused event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUnpausedEvent struct {
	Account common.Address
}

// UpdateFeeVault represents a UpdateFeeVault event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUpdateFeeVaultEvent struct {
	OldFeeVault common.Address
	NewFeeVault common.Address
}

// UpdateMaxReplayTimes represents a UpdateMaxReplayTimes event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUpdateMaxReplayTimesEvent struct {
	MaxReplayTimes *big.Int
}
