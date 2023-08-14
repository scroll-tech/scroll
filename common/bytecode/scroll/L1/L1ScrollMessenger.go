// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package L1

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

// IL1ScrollMessengerL2MessageProof is an auto generated low-level Go binding around an user-defined struct.
type IL1ScrollMessengerL2MessageProof struct {
	BatchIndex  *big.Int
	MerkleProof []byte
}

// L1ScrollMessengerMetaData contains all meta data concerning the L1ScrollMessenger contract.
var (
	L1ScrollMessengerMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\",\"indexed\":true}],\"type\":\"event\",\"name\":\"FailedRelayedMessage\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Initialized\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Paused\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\",\"indexed\":true}],\"type\":\"event\",\"name\":\"RelayedMessage\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\",\"indexed\":false}],\"type\":\"event\",\"name\":\"SentMessage\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Unpaused\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oldFeeVault\",\"type\":\"address\",\"indexed\":false},{\"internalType\":\"address\",\"name\":\"_newFeeVault\",\"type\":\"address\",\"indexed\":false}],\"type\":\"event\",\"name\":\"UpdateFeeVault\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"oldMaxReplayTimes\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"newMaxReplayTimes\",\"type\":\"uint256\",\"indexed\":false}],\"type\":\"event\",\"name\":\"UpdateMaxReplayTimes\",\"anonymous\":false},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"counterpart\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_messageNonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"dropMessage\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"feeVault\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_counterpart\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_feeVault\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_rollup\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_messageQueue\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"initialize\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"isL1MessageDropped\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}]},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"isL1MessageSent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}]},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"isL2MessageExecuted\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"maxReplayTimes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"messageQueue\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}]},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"prevReplayIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"structIL1ScrollMessenger.L2MessageProof\",\"name\":\"_proof\",\"type\":\"tuple\",\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"merkleProof\",\"type\":\"bytes\"}]}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"relayMessageWithProof\"},{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"renounceOwnership\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_messageNonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint32\",\"name\":\"_newGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"address\",\"name\":\"_refundAddress\",\"type\":\"address\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"replayMessage\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"replayStates\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"times\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"lastIndex\",\"type\":\"uint128\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_refundAddress\",\"type\":\"address\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"sendMessage\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"function\",\"name\":\"sendMessage\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_status\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setPause\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"transferOwnership\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newFeeVault\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"updateFeeVault\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_newMaxReplayTimes\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"updateMaxReplayTimes\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"xDomainMessageSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
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

// L1ScrollMessenger is an auto generated Go binding around an Ethereum contract.
type L1ScrollMessenger struct {
	Address common.Address // contract address
	ABI     *abi.ABI       // L1ScrollMessengerABI is the input ABI used to generate the binding from.
	Parsers map[common.Hash]func(log *types.Log) (interface{}, error)

	L1ScrollMessengerCaller     // Read-only binding to the contract
	L1ScrollMessengerTransactor // Write-only binding to the contract
}

// GetAddress return L1ScrollMessenger's contract address.
func (o *L1ScrollMessenger) GetAddress() common.Address {
	return o.Address
}

// GetParsers return Parsers
func (o *L1ScrollMessenger) GetParsers() map[common.Hash]func(log *types.Log) (interface{}, error) {
	return o.Parsers
}

// GetABI return *big.ABI
func (o *L1ScrollMessenger) GetABI() *abi.ABI {
	return o.ABI
}

// L1ScrollMessengerCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1ScrollMessengerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ScrollMessengerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1ScrollMessengerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NewL1ScrollMessenger creates a new instance of L1ScrollMessenger, bound to a specific deployed contract.
func NewL1ScrollMessenger(address common.Address, backend bind.ContractBackend) (*L1ScrollMessenger, error) {
	contract, err := bindL1ScrollMessenger(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	sigAbi, err := L1ScrollMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	var parsers = map[common.Hash]func(log *types.Log) (interface{}, error){}

	parsers[sigAbi.Events["FailedRelayedMessage"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1ScrollMessengerFailedRelayedMessageEvent)
		if err := contract.UnpackLog(event, "FailedRelayedMessage", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Initialized"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1ScrollMessengerInitializedEvent)
		if err := contract.UnpackLog(event, "Initialized", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["OwnershipTransferred"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1ScrollMessengerOwnershipTransferredEvent)
		if err := contract.UnpackLog(event, "OwnershipTransferred", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Paused"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1ScrollMessengerPausedEvent)
		if err := contract.UnpackLog(event, "Paused", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["RelayedMessage"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1ScrollMessengerRelayedMessageEvent)
		if err := contract.UnpackLog(event, "RelayedMessage", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["SentMessage"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1ScrollMessengerSentMessageEvent)
		if err := contract.UnpackLog(event, "SentMessage", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Unpaused"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1ScrollMessengerUnpausedEvent)
		if err := contract.UnpackLog(event, "Unpaused", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["UpdateFeeVault"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1ScrollMessengerUpdateFeeVaultEvent)
		if err := contract.UnpackLog(event, "UpdateFeeVault", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["UpdateMaxReplayTimes"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1ScrollMessengerUpdateMaxReplayTimesEvent)
		if err := contract.UnpackLog(event, "UpdateMaxReplayTimes", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	return &L1ScrollMessenger{ABI: sigAbi, Address: address, Parsers: parsers, L1ScrollMessengerCaller: L1ScrollMessengerCaller{contract: contract}, L1ScrollMessengerTransactor: L1ScrollMessengerTransactor{contract: contract}}, nil
}

// NewL1ScrollMessengerCaller creates a new read-only instance of L1ScrollMessenger, bound to a specific deployed contract.
func NewL1ScrollMessengerCaller(address common.Address, caller bind.ContractCaller) (*L1ScrollMessengerCaller, error) {
	contract, err := bindL1ScrollMessenger(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerCaller{contract: contract}, nil
}

// NewL1ScrollMessengerTransactor creates a new write-only instance of L1ScrollMessenger, bound to a specific deployed contract.
func NewL1ScrollMessengerTransactor(address common.Address, transactor bind.ContractTransactor) (*L1ScrollMessengerTransactor, error) {
	contract, err := bindL1ScrollMessenger(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerTransactor{contract: contract}, nil
}

// bindL1ScrollMessenger binds a generic wrapper to an already deployed contract.
func bindL1ScrollMessenger(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1ScrollMessengerMetaData.ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Counterpart is a free data retrieval call binding the contract method 0x797594b0.
//
// Solidity: function counterpart() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) Counterpart(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "counterpart")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FeeVault is a free data retrieval call binding the contract method 0x478222c2.
//
// Solidity: function feeVault() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) FeeVault(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "feeVault")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// IsL1MessageDropped is a free data retrieval call binding the contract method 0xb604bf4c.
//
// Solidity: function isL1MessageDropped(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) IsL1MessageDropped(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "isL1MessageDropped", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsL1MessageSent is a free data retrieval call binding the contract method 0x69058083.
//
// Solidity: function isL1MessageSent(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) IsL1MessageSent(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "isL1MessageSent", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsL2MessageExecuted is a free data retrieval call binding the contract method 0x088681a7.
//
// Solidity: function isL2MessageExecuted(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) IsL2MessageExecuted(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "isL2MessageExecuted", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// MaxReplayTimes is a free data retrieval call binding the contract method 0x946130d8.
//
// Solidity: function maxReplayTimes() view returns(uint256)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) MaxReplayTimes(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "maxReplayTimes")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MessageQueue is a free data retrieval call binding the contract method 0x3b70c18a.
//
// Solidity: function messageQueue() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) MessageQueue(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "messageQueue")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// PrevReplayIndex is a free data retrieval call binding the contract method 0xea7ec514.
//
// Solidity: function prevReplayIndex(uint256 ) view returns(uint256)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) PrevReplayIndex(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "prevReplayIndex", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ReplayStates is a free data retrieval call binding the contract method 0x846d4d7a.
//
// Solidity: function replayStates(bytes32 ) view returns(uint128 times, uint128 lastIndex)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) ReplayStates(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Times     *big.Int
	LastIndex *big.Int
}, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "replayStates", arg0)

	outstruct := new(struct {
		Times     *big.Int
		LastIndex *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Times = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.LastIndex = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) XDomainMessageSender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "xDomainMessageSender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DropMessage is a paid mutator transaction binding the contract method 0x29907acd.
//
// Solidity: function dropMessage(address _from, address _to, uint256 _value, uint256 _messageNonce, bytes _message) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) DropMessage(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int, _messageNonce *big.Int, _message []byte) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "dropMessage", _from, _to, _value, _messageNonce, _message)
}

// Initialize is a paid mutator transaction binding the contract method 0xf8c8765e.
//
// Solidity: function initialize(address _counterpart, address _feeVault, address _rollup, address _messageQueue) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) Initialize(opts *bind.TransactOpts, _counterpart common.Address, _feeVault common.Address, _rollup common.Address, _messageQueue common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "initialize", _counterpart, _feeVault, _rollup, _messageQueue)
}

// RelayMessageWithProof is a paid mutator transaction binding the contract method 0xc311b6fc.
//
// Solidity: function relayMessageWithProof(address _from, address _to, uint256 _value, uint256 _nonce, bytes _message, (uint256,bytes) _proof) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) RelayMessageWithProof(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int, _nonce *big.Int, _message []byte, _proof IL1ScrollMessengerL2MessageProof) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "relayMessageWithProof", _from, _to, _value, _nonce, _message, _proof)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "renounceOwnership")
}

// ReplayMessage is a paid mutator transaction binding the contract method 0x55004105.
//
// Solidity: function replayMessage(address _from, address _to, uint256 _value, uint256 _messageNonce, bytes _message, uint32 _newGasLimit, address _refundAddress) payable returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) ReplayMessage(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int, _messageNonce *big.Int, _message []byte, _newGasLimit uint32, _refundAddress common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "replayMessage", _from, _to, _value, _messageNonce, _message, _newGasLimit, _refundAddress)
}

// SendMessage is a paid mutator transaction binding the contract method 0x5f7b1577.
//
// Solidity: function sendMessage(address _to, uint256 _value, bytes _message, uint256 _gasLimit, address _refundAddress) payable returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) SendMessage(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _message []byte, _gasLimit *big.Int, _refundAddress common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "sendMessage", _to, _value, _message, _gasLimit, _refundAddress)
}

// SendMessage0 is a paid mutator transaction binding the contract method 0xb2267a7b.
//
// Solidity: function sendMessage(address _to, uint256 _value, bytes _message, uint256 _gasLimit) payable returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) SendMessage0(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _message []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "sendMessage0", _to, _value, _message, _gasLimit)
}

// SetPause is a paid mutator transaction binding the contract method 0xbedb86fb.
//
// Solidity: function setPause(bool _status) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) SetPause(opts *bind.TransactOpts, _status bool) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "setPause", _status)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "transferOwnership", newOwner)
}

// UpdateFeeVault is a paid mutator transaction binding the contract method 0x2a6cccb2.
//
// Solidity: function updateFeeVault(address _newFeeVault) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) UpdateFeeVault(opts *bind.TransactOpts, _newFeeVault common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "updateFeeVault", _newFeeVault)
}

// UpdateMaxReplayTimes is a paid mutator transaction binding the contract method 0x407c1955.
//
// Solidity: function updateMaxReplayTimes(uint256 _newMaxReplayTimes) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) UpdateMaxReplayTimes(opts *bind.TransactOpts, _newMaxReplayTimes *big.Int) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "updateMaxReplayTimes", _newMaxReplayTimes)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// L1ScrollMessengerFailedRelayedMessage represents a FailedRelayedMessage event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerFailedRelayedMessageEvent struct {
	MessageHash [32]byte
	raw         *types.Log // Blockchain specific contextual infos
}

// L1ScrollMessengerInitialized represents a Initialized event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerInitializedEvent struct {
	Version uint8
	raw     *types.Log // Blockchain specific contextual infos
}

// L1ScrollMessengerOwnershipTransferred represents a OwnershipTransferred event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	raw           *types.Log // Blockchain specific contextual infos
}

// L1ScrollMessengerPaused represents a Paused event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerPausedEvent struct {
	Account common.Address
	raw     *types.Log // Blockchain specific contextual infos
}

// L1ScrollMessengerRelayedMessage represents a RelayedMessage event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerRelayedMessageEvent struct {
	MessageHash [32]byte
	raw         *types.Log // Blockchain specific contextual infos
}

// L1ScrollMessengerSentMessage represents a SentMessage event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerSentMessageEvent struct {
	Sender       common.Address
	Target       common.Address
	Value        *big.Int
	MessageNonce *big.Int
	GasLimit     *big.Int
	Message      []byte
	raw          *types.Log // Blockchain specific contextual infos
}

// L1ScrollMessengerUnpaused represents a Unpaused event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUnpausedEvent struct {
	Account common.Address
	raw     *types.Log // Blockchain specific contextual infos
}

// L1ScrollMessengerUpdateFeeVault represents a UpdateFeeVault event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUpdateFeeVaultEvent struct {
	OldFeeVault common.Address
	NewFeeVault common.Address
	raw         *types.Log // Blockchain specific contextual infos
}

// L1ScrollMessengerUpdateMaxReplayTimes represents a UpdateMaxReplayTimes event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUpdateMaxReplayTimesEvent struct {
	OldMaxReplayTimes *big.Int
	NewMaxReplayTimes *big.Int
	raw               *types.Log // Blockchain specific contextual infos
}
