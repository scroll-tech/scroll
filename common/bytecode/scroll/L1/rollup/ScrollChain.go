// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package rollup

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

// ScrollChainMetaData contains all meta data concerning the ScrollChain contract.
var (
	ScrollChainMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_chainId\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\",\"indexed\":true},{\"internalType\":\"bytes32\",\"name\":\"batchHash\",\"type\":\"bytes32\",\"indexed\":true}],\"type\":\"event\",\"name\":\"CommitBatch\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\",\"indexed\":true},{\"internalType\":\"bytes32\",\"name\":\"batchHash\",\"type\":\"bytes32\",\"indexed\":true},{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\",\"indexed\":false},{\"internalType\":\"bytes32\",\"name\":\"withdrawRoot\",\"type\":\"bytes32\",\"indexed\":false}],\"type\":\"event\",\"name\":\"FinalizeBatch\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Initialized\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Paused\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\",\"indexed\":true},{\"internalType\":\"bytes32\",\"name\":\"batchHash\",\"type\":\"bytes32\",\"indexed\":true}],\"type\":\"event\",\"name\":\"RevertBatch\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\",\"indexed\":false}],\"type\":\"event\",\"name\":\"Unpaused\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"oldMaxNumL2TxInChunk\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"newMaxNumL2TxInChunk\",\"type\":\"uint256\",\"indexed\":false}],\"type\":\"event\",\"name\":\"UpdateMaxNumL2TxInChunk\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"bool\",\"name\":\"status\",\"type\":\"bool\",\"indexed\":false}],\"type\":\"event\",\"name\":\"UpdateProver\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"bool\",\"name\":\"status\",\"type\":\"bool\",\"indexed\":false}],\"type\":\"event\",\"name\":\"UpdateSequencer\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"oldVerifier\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"newVerifier\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"UpdateVerifier\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"addProver\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"addSequencer\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"_version\",\"type\":\"uint8\"},{\"internalType\":\"bytes\",\"name\":\"_parentBatchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes[]\",\"name\":\"_chunks\",\"type\":\"bytes[]\"},{\"internalType\":\"bytes\",\"name\":\"_skippedL1MessageBitmap\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"commitBatch\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"committedBatches\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}]},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"_prevStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_postStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_withdrawRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"_aggrProof\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"finalizeBatchWithProof\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"finalizedStateRoots\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}]},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"_stateRoot\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"importGenesisBatch\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_messageQueue\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_verifier\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_maxNumL2TxInChunk\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"initialize\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"isBatchFinalized\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"isProver\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"isSequencer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"lastFinalizedBatchIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"layer2ChainId\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"maxNumL2TxInChunk\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"messageQueue\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"removeProver\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"removeSequencer\"},{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"renounceOwnership\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_count\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"revertBatch\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_status\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"setPause\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"transferOwnership\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_maxNumL2TxInChunk\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"updateMaxNumL2TxInChunk\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newVerifier\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"updateVerifier\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"verifier\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"withdrawRoots\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}]}]",
	}
	// ScrollChainABI is the input ABI used to generate the binding from.
	ScrollChainABI *abi.ABI

	// CommitBatch event
	ScrollChainCommitBatchEventSignature common.Hash

	// FinalizeBatch event
	ScrollChainFinalizeBatchEventSignature common.Hash

	// Initialized event
	ScrollChainInitializedEventSignature common.Hash

	// OwnershipTransferred event
	ScrollChainOwnershipTransferredEventSignature common.Hash

	// Paused event
	ScrollChainPausedEventSignature common.Hash

	// RevertBatch event
	ScrollChainRevertBatchEventSignature common.Hash

	// Unpaused event
	ScrollChainUnpausedEventSignature common.Hash

	// UpdateMaxNumL2TxInChunk event
	ScrollChainUpdateMaxNumL2TxInChunkEventSignature common.Hash

	// UpdateProver event
	ScrollChainUpdateProverEventSignature common.Hash

	// UpdateSequencer event
	ScrollChainUpdateSequencerEventSignature common.Hash

	// UpdateVerifier event
	ScrollChainUpdateVerifierEventSignature common.Hash
)

func init() {
	sigAbi, err := ScrollChainMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	ScrollChainABI = sigAbi

	// CommitBatch event
	ScrollChainCommitBatchEventSignature = sigAbi.Events["CommitBatch"].ID

	// FinalizeBatch event
	ScrollChainFinalizeBatchEventSignature = sigAbi.Events["FinalizeBatch"].ID

	// Initialized event
	ScrollChainInitializedEventSignature = sigAbi.Events["Initialized"].ID

	// OwnershipTransferred event
	ScrollChainOwnershipTransferredEventSignature = sigAbi.Events["OwnershipTransferred"].ID

	// Paused event
	ScrollChainPausedEventSignature = sigAbi.Events["Paused"].ID

	// RevertBatch event
	ScrollChainRevertBatchEventSignature = sigAbi.Events["RevertBatch"].ID

	// Unpaused event
	ScrollChainUnpausedEventSignature = sigAbi.Events["Unpaused"].ID

	// UpdateMaxNumL2TxInChunk event
	ScrollChainUpdateMaxNumL2TxInChunkEventSignature = sigAbi.Events["UpdateMaxNumL2TxInChunk"].ID

	// UpdateProver event
	ScrollChainUpdateProverEventSignature = sigAbi.Events["UpdateProver"].ID

	// UpdateSequencer event
	ScrollChainUpdateSequencerEventSignature = sigAbi.Events["UpdateSequencer"].ID

	// UpdateVerifier event
	ScrollChainUpdateVerifierEventSignature = sigAbi.Events["UpdateVerifier"].ID

}

// ScrollChain is an auto generated Go binding around an Ethereum contract.
type ScrollChain struct {
	Address common.Address // contract address
	ABI     *abi.ABI       // ScrollChainABI is the input ABI used to generate the binding from.
	Parsers map[common.Hash]func(log *types.Log) (interface{}, error)

	ScrollChainCaller     // Read-only binding to the contract
	ScrollChainTransactor // Write-only binding to the contract
}

// GetAddress return ScrollChain's contract address.
func (o *ScrollChain) GetAddress() common.Address {
	return o.Address
}

// GetParsers return Parsers
func (o *ScrollChain) GetParsers() map[common.Hash]func(log *types.Log) (interface{}, error) {
	return o.Parsers
}

// GetABI return *big.ABI
func (o *ScrollChain) GetABI() *abi.ABI {
	return o.ABI
}

// ScrollChainCaller is an auto generated read-only Go binding around an Ethereum contract.
type ScrollChainCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ScrollChainTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ScrollChainTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NewScrollChain creates a new instance of ScrollChain, bound to a specific deployed contract.
func NewScrollChain(address common.Address, backend bind.ContractBackend) (*ScrollChain, error) {
	contract, err := bindScrollChain(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	sigAbi, err := ScrollChainMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	var parsers = map[common.Hash]func(log *types.Log) (interface{}, error){}

	parsers[sigAbi.Events["CommitBatch"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(ScrollChainCommitBatchEvent)
		if err := contract.UnpackLog(event, "CommitBatch", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["FinalizeBatch"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(ScrollChainFinalizeBatchEvent)
		if err := contract.UnpackLog(event, "FinalizeBatch", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Initialized"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(ScrollChainInitializedEvent)
		if err := contract.UnpackLog(event, "Initialized", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["OwnershipTransferred"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(ScrollChainOwnershipTransferredEvent)
		if err := contract.UnpackLog(event, "OwnershipTransferred", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Paused"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(ScrollChainPausedEvent)
		if err := contract.UnpackLog(event, "Paused", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["RevertBatch"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(ScrollChainRevertBatchEvent)
		if err := contract.UnpackLog(event, "RevertBatch", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["Unpaused"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(ScrollChainUnpausedEvent)
		if err := contract.UnpackLog(event, "Unpaused", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["UpdateMaxNumL2TxInChunk"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(ScrollChainUpdateMaxNumL2TxInChunkEvent)
		if err := contract.UnpackLog(event, "UpdateMaxNumL2TxInChunk", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["UpdateProver"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(ScrollChainUpdateProverEvent)
		if err := contract.UnpackLog(event, "UpdateProver", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["UpdateSequencer"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(ScrollChainUpdateSequencerEvent)
		if err := contract.UnpackLog(event, "UpdateSequencer", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["UpdateVerifier"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(ScrollChainUpdateVerifierEvent)
		if err := contract.UnpackLog(event, "UpdateVerifier", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	return &ScrollChain{ABI: sigAbi, Address: address, Parsers: parsers, ScrollChainCaller: ScrollChainCaller{contract: contract}, ScrollChainTransactor: ScrollChainTransactor{contract: contract}}, nil
}

// NewScrollChainCaller creates a new read-only instance of ScrollChain, bound to a specific deployed contract.
func NewScrollChainCaller(address common.Address, caller bind.ContractCaller) (*ScrollChainCaller, error) {
	contract, err := bindScrollChain(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ScrollChainCaller{contract: contract}, nil
}

// NewScrollChainTransactor creates a new write-only instance of ScrollChain, bound to a specific deployed contract.
func NewScrollChainTransactor(address common.Address, transactor bind.ContractTransactor) (*ScrollChainTransactor, error) {
	contract, err := bindScrollChain(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ScrollChainTransactor{contract: contract}, nil
}

// bindScrollChain binds a generic wrapper to an already deployed contract.
func bindScrollChain(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ScrollChainMetaData.ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// CommittedBatches is a free data retrieval call binding the contract method 0x2362f03e.
//
// Solidity: function committedBatches(uint256 ) view returns(bytes32)
func (_ScrollChain *ScrollChainCaller) CommittedBatches(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "committedBatches", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// FinalizedStateRoots is a free data retrieval call binding the contract method 0x2571098d.
//
// Solidity: function finalizedStateRoots(uint256 ) view returns(bytes32)
func (_ScrollChain *ScrollChainCaller) FinalizedStateRoots(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "finalizedStateRoots", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// IsBatchFinalized is a free data retrieval call binding the contract method 0x116a1f42.
//
// Solidity: function isBatchFinalized(uint256 _batchIndex) view returns(bool)
func (_ScrollChain *ScrollChainCaller) IsBatchFinalized(opts *bind.CallOpts, _batchIndex *big.Int) (bool, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "isBatchFinalized", _batchIndex)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsProver is a free data retrieval call binding the contract method 0x0a245924.
//
// Solidity: function isProver(address ) view returns(bool)
func (_ScrollChain *ScrollChainCaller) IsProver(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "isProver", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsSequencer is a free data retrieval call binding the contract method 0x6d46e987.
//
// Solidity: function isSequencer(address ) view returns(bool)
func (_ScrollChain *ScrollChainCaller) IsSequencer(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "isSequencer", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// LastFinalizedBatchIndex is a free data retrieval call binding the contract method 0x059def61.
//
// Solidity: function lastFinalizedBatchIndex() view returns(uint256)
func (_ScrollChain *ScrollChainCaller) LastFinalizedBatchIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "lastFinalizedBatchIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Layer2ChainId is a free data retrieval call binding the contract method 0x03c7f4af.
//
// Solidity: function layer2ChainId() view returns(uint64)
func (_ScrollChain *ScrollChainCaller) Layer2ChainId(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "layer2ChainId")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MaxNumL2TxInChunk is a free data retrieval call binding the contract method 0xd19a92d7.
//
// Solidity: function maxNumL2TxInChunk() view returns(uint256)
func (_ScrollChain *ScrollChainCaller) MaxNumL2TxInChunk(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "maxNumL2TxInChunk")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MessageQueue is a free data retrieval call binding the contract method 0x3b70c18a.
//
// Solidity: function messageQueue() view returns(address)
func (_ScrollChain *ScrollChainCaller) MessageQueue(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "messageQueue")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ScrollChain *ScrollChainCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_ScrollChain *ScrollChainCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_ScrollChain *ScrollChainCaller) Verifier(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "verifier")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// WithdrawRoots is a free data retrieval call binding the contract method 0xea5f084f.
//
// Solidity: function withdrawRoots(uint256 ) view returns(bytes32)
func (_ScrollChain *ScrollChainCaller) WithdrawRoots(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _ScrollChain.contract.Call(opts, &out, "withdrawRoots", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// AddProver is a paid mutator transaction binding the contract method 0x1d49e457.
//
// Solidity: function addProver(address _account) returns()
func (_ScrollChain *ScrollChainTransactor) AddProver(opts *bind.TransactOpts, _account common.Address) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "addProver", _account)
}

// AddSequencer is a paid mutator transaction binding the contract method 0x8a336231.
//
// Solidity: function addSequencer(address _account) returns()
func (_ScrollChain *ScrollChainTransactor) AddSequencer(opts *bind.TransactOpts, _account common.Address) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "addSequencer", _account)
}

// CommitBatch is a paid mutator transaction binding the contract method 0x1325aca0.
//
// Solidity: function commitBatch(uint8 _version, bytes _parentBatchHeader, bytes[] _chunks, bytes _skippedL1MessageBitmap) returns()
func (_ScrollChain *ScrollChainTransactor) CommitBatch(opts *bind.TransactOpts, _version uint8, _parentBatchHeader []byte, _chunks [][]byte, _skippedL1MessageBitmap []byte) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "commitBatch", _version, _parentBatchHeader, _chunks, _skippedL1MessageBitmap)
}

// FinalizeBatchWithProof is a paid mutator transaction binding the contract method 0x31fa742d.
//
// Solidity: function finalizeBatchWithProof(bytes _batchHeader, bytes32 _prevStateRoot, bytes32 _postStateRoot, bytes32 _withdrawRoot, bytes _aggrProof) returns()
func (_ScrollChain *ScrollChainTransactor) FinalizeBatchWithProof(opts *bind.TransactOpts, _batchHeader []byte, _prevStateRoot [32]byte, _postStateRoot [32]byte, _withdrawRoot [32]byte, _aggrProof []byte) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "finalizeBatchWithProof", _batchHeader, _prevStateRoot, _postStateRoot, _withdrawRoot, _aggrProof)
}

// ImportGenesisBatch is a paid mutator transaction binding the contract method 0x3fdeecb2.
//
// Solidity: function importGenesisBatch(bytes _batchHeader, bytes32 _stateRoot) returns()
func (_ScrollChain *ScrollChainTransactor) ImportGenesisBatch(opts *bind.TransactOpts, _batchHeader []byte, _stateRoot [32]byte) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "importGenesisBatch", _batchHeader, _stateRoot)
}

// Initialize is a paid mutator transaction binding the contract method 0x1794bb3c.
//
// Solidity: function initialize(address _messageQueue, address _verifier, uint256 _maxNumL2TxInChunk) returns()
func (_ScrollChain *ScrollChainTransactor) Initialize(opts *bind.TransactOpts, _messageQueue common.Address, _verifier common.Address, _maxNumL2TxInChunk *big.Int) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "initialize", _messageQueue, _verifier, _maxNumL2TxInChunk)
}

// RemoveProver is a paid mutator transaction binding the contract method 0xb571d3dd.
//
// Solidity: function removeProver(address _account) returns()
func (_ScrollChain *ScrollChainTransactor) RemoveProver(opts *bind.TransactOpts, _account common.Address) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "removeProver", _account)
}

// RemoveSequencer is a paid mutator transaction binding the contract method 0x6989ca7c.
//
// Solidity: function removeSequencer(address _account) returns()
func (_ScrollChain *ScrollChainTransactor) RemoveSequencer(opts *bind.TransactOpts, _account common.Address) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "removeSequencer", _account)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ScrollChain *ScrollChainTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "renounceOwnership")
}

// RevertBatch is a paid mutator transaction binding the contract method 0x10d44583.
//
// Solidity: function revertBatch(bytes _batchHeader, uint256 _count) returns()
func (_ScrollChain *ScrollChainTransactor) RevertBatch(opts *bind.TransactOpts, _batchHeader []byte, _count *big.Int) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "revertBatch", _batchHeader, _count)
}

// SetPause is a paid mutator transaction binding the contract method 0xbedb86fb.
//
// Solidity: function setPause(bool _status) returns()
func (_ScrollChain *ScrollChainTransactor) SetPause(opts *bind.TransactOpts, _status bool) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "setPause", _status)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ScrollChain *ScrollChainTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "transferOwnership", newOwner)
}

// UpdateMaxNumL2TxInChunk is a paid mutator transaction binding the contract method 0x7bb67dde.
//
// Solidity: function updateMaxNumL2TxInChunk(uint256 _maxNumL2TxInChunk) returns()
func (_ScrollChain *ScrollChainTransactor) UpdateMaxNumL2TxInChunk(opts *bind.TransactOpts, _maxNumL2TxInChunk *big.Int) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "updateMaxNumL2TxInChunk", _maxNumL2TxInChunk)
}

// UpdateVerifier is a paid mutator transaction binding the contract method 0x97fc007c.
//
// Solidity: function updateVerifier(address _newVerifier) returns()
func (_ScrollChain *ScrollChainTransactor) UpdateVerifier(opts *bind.TransactOpts, _newVerifier common.Address) (*types.Transaction, error) {
	return _ScrollChain.contract.Transact(opts, "updateVerifier", _newVerifier)
}

// ScrollChainCommitBatch represents a CommitBatch event raised by the ScrollChain contract.
type ScrollChainCommitBatchEvent struct {
	BatchIndex *big.Int
	BatchHash  [32]byte
	raw        *types.Log // Blockchain specific contextual infos
}

// ScrollChainFinalizeBatch represents a FinalizeBatch event raised by the ScrollChain contract.
type ScrollChainFinalizeBatchEvent struct {
	BatchIndex   *big.Int
	BatchHash    [32]byte
	StateRoot    [32]byte
	WithdrawRoot [32]byte
	raw          *types.Log // Blockchain specific contextual infos
}

// ScrollChainInitialized represents a Initialized event raised by the ScrollChain contract.
type ScrollChainInitializedEvent struct {
	Version uint8
	raw     *types.Log // Blockchain specific contextual infos
}

// ScrollChainOwnershipTransferred represents a OwnershipTransferred event raised by the ScrollChain contract.
type ScrollChainOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	raw           *types.Log // Blockchain specific contextual infos
}

// ScrollChainPaused represents a Paused event raised by the ScrollChain contract.
type ScrollChainPausedEvent struct {
	Account common.Address
	raw     *types.Log // Blockchain specific contextual infos
}

// ScrollChainRevertBatch represents a RevertBatch event raised by the ScrollChain contract.
type ScrollChainRevertBatchEvent struct {
	BatchIndex *big.Int
	BatchHash  [32]byte
	raw        *types.Log // Blockchain specific contextual infos
}

// ScrollChainUnpaused represents a Unpaused event raised by the ScrollChain contract.
type ScrollChainUnpausedEvent struct {
	Account common.Address
	raw     *types.Log // Blockchain specific contextual infos
}

// ScrollChainUpdateMaxNumL2TxInChunk represents a UpdateMaxNumL2TxInChunk event raised by the ScrollChain contract.
type ScrollChainUpdateMaxNumL2TxInChunkEvent struct {
	OldMaxNumL2TxInChunk *big.Int
	NewMaxNumL2TxInChunk *big.Int
	raw                  *types.Log // Blockchain specific contextual infos
}

// ScrollChainUpdateProver represents a UpdateProver event raised by the ScrollChain contract.
type ScrollChainUpdateProverEvent struct {
	Account common.Address
	Status  bool
	raw     *types.Log // Blockchain specific contextual infos
}

// ScrollChainUpdateSequencer represents a UpdateSequencer event raised by the ScrollChain contract.
type ScrollChainUpdateSequencerEvent struct {
	Account common.Address
	Status  bool
	raw     *types.Log // Blockchain specific contextual infos
}

// ScrollChainUpdateVerifier represents a UpdateVerifier event raised by the ScrollChain contract.
type ScrollChainUpdateVerifierEvent struct {
	OldVerifier common.Address
	NewVerifier common.Address
	raw         *types.Log // Blockchain specific contextual infos
}
