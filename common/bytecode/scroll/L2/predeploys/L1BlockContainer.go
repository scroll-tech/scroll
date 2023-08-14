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

// L1BlockContainerMetaData contains all meta data concerning the L1BlockContainer contract.
var (
	L1BlockContainerMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\",\"indexed\":true},{\"internalType\":\"uint256\",\"name\":\"blockHeight\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"blockTimestamp\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\",\"indexed\":false},{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\",\"indexed\":false}],\"type\":\"event\",\"name\":\"ImportBlock\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oldOwner\",\"type\":\"address\",\"indexed\":true},{\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\",\"indexed\":true}],\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oldWhitelist\",\"type\":\"address\",\"indexed\":false},{\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\",\"indexed\":false}],\"type\":\"event\",\"name\":\"UpdateWhitelist\",\"anonymous\":false},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_blockHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"getBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_blockHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"getStateRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}]},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_blockHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"_blockHeaderRLP\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"_updateGasPriceOracle\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"importBlockHeader\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_startBlockHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"_startBlockHeight\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_startBlockTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"_startBlockBaseFee\",\"type\":\"uint128\"},{\"internalType\":\"bytes32\",\"name\":\"_startStateRoot\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"initialize\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"latestBaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"latestBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"latestBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"latestBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}]},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"metadata\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"height\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"baseFee\",\"type\":\"uint128\"}]},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}]},{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"renounceOwnership\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"stateRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}]},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"transferOwnership\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"updateWhitelist\"},{\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"whitelist\",\"outputs\":[{\"internalType\":\"contractIWhitelist\",\"name\":\"\",\"type\":\"address\"}]}]",
	}
	// L1BlockContainerABI is the input ABI used to generate the binding from.
	L1BlockContainerABI *abi.ABI

	// ImportBlock event
	L1BlockContainerImportBlockEventSignature common.Hash

	// OwnershipTransferred event
	L1BlockContainerOwnershipTransferredEventSignature common.Hash

	// UpdateWhitelist event
	L1BlockContainerUpdateWhitelistEventSignature common.Hash
)

func init() {
	sigAbi, err := L1BlockContainerMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	L1BlockContainerABI = sigAbi

	// ImportBlock event
	L1BlockContainerImportBlockEventSignature = sigAbi.Events["ImportBlock"].ID

	// OwnershipTransferred event
	L1BlockContainerOwnershipTransferredEventSignature = sigAbi.Events["OwnershipTransferred"].ID

	// UpdateWhitelist event
	L1BlockContainerUpdateWhitelistEventSignature = sigAbi.Events["UpdateWhitelist"].ID

}

// L1BlockContainer is an auto generated Go binding around an Ethereum contract.
type L1BlockContainer struct {
	Address common.Address // contract address
	ABI     *abi.ABI       // L1BlockContainerABI is the input ABI used to generate the binding from.
	Parsers map[common.Hash]func(log *types.Log) (interface{}, error)

	L1BlockContainerCaller     // Read-only binding to the contract
	L1BlockContainerTransactor // Write-only binding to the contract
}

// GetAddress return L1BlockContainer's contract address.
func (o *L1BlockContainer) GetAddress() common.Address {
	return o.Address
}

// GetParsers return Parsers
func (o *L1BlockContainer) GetParsers() map[common.Hash]func(log *types.Log) (interface{}, error) {
	return o.Parsers
}

// GetABI return *big.ABI
func (o *L1BlockContainer) GetABI() *abi.ABI {
	return o.ABI
}

// L1BlockContainerCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1BlockContainerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockContainerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1BlockContainerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NewL1BlockContainer creates a new instance of L1BlockContainer, bound to a specific deployed contract.
func NewL1BlockContainer(address common.Address, backend bind.ContractBackend) (*L1BlockContainer, error) {
	contract, err := bindL1BlockContainer(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	sigAbi, err := L1BlockContainerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	var parsers = map[common.Hash]func(log *types.Log) (interface{}, error){}

	parsers[sigAbi.Events["ImportBlock"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1BlockContainerImportBlockEvent)
		if err := contract.UnpackLog(event, "ImportBlock", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["OwnershipTransferred"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1BlockContainerOwnershipTransferredEvent)
		if err := contract.UnpackLog(event, "OwnershipTransferred", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	parsers[sigAbi.Events["UpdateWhitelist"].ID] = func(log *types.Log) (interface{}, error) {
		event := new(L1BlockContainerUpdateWhitelistEvent)
		if err := contract.UnpackLog(event, "UpdateWhitelist", *log); err != nil {
			return nil, err
		}
		return event, nil
	}

	return &L1BlockContainer{ABI: sigAbi, Address: address, Parsers: parsers, L1BlockContainerCaller: L1BlockContainerCaller{contract: contract}, L1BlockContainerTransactor: L1BlockContainerTransactor{contract: contract}}, nil
}

// NewL1BlockContainerCaller creates a new read-only instance of L1BlockContainer, bound to a specific deployed contract.
func NewL1BlockContainerCaller(address common.Address, caller bind.ContractCaller) (*L1BlockContainerCaller, error) {
	contract, err := bindL1BlockContainer(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlockContainerCaller{contract: contract}, nil
}

// NewL1BlockContainerTransactor creates a new write-only instance of L1BlockContainer, bound to a specific deployed contract.
func NewL1BlockContainerTransactor(address common.Address, transactor bind.ContractTransactor) (*L1BlockContainerTransactor, error) {
	contract, err := bindL1BlockContainer(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlockContainerTransactor{contract: contract}, nil
}

// bindL1BlockContainer binds a generic wrapper to an already deployed contract.
func bindL1BlockContainer(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1BlockContainerMetaData.ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// GetBlockTimestamp is a free data retrieval call binding the contract method 0x56e214e4.
//
// Solidity: function getBlockTimestamp(bytes32 _blockHash) view returns(uint256)
func (_L1BlockContainer *L1BlockContainerCaller) GetBlockTimestamp(opts *bind.CallOpts, _blockHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _L1BlockContainer.contract.Call(opts, &out, "getBlockTimestamp", _blockHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetStateRoot is a free data retrieval call binding the contract method 0x2cb58378.
//
// Solidity: function getStateRoot(bytes32 _blockHash) view returns(bytes32)
func (_L1BlockContainer *L1BlockContainerCaller) GetStateRoot(opts *bind.CallOpts, _blockHash [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _L1BlockContainer.contract.Call(opts, &out, "getStateRoot", _blockHash)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// LatestBaseFee is a free data retrieval call binding the contract method 0x0385f4f1.
//
// Solidity: function latestBaseFee() view returns(uint256)
func (_L1BlockContainer *L1BlockContainerCaller) LatestBaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1BlockContainer.contract.Call(opts, &out, "latestBaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBlockHash is a free data retrieval call binding the contract method 0x6c4f6ba9.
//
// Solidity: function latestBlockHash() view returns(bytes32)
func (_L1BlockContainer *L1BlockContainerCaller) LatestBlockHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1BlockContainer.contract.Call(opts, &out, "latestBlockHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L1BlockContainer *L1BlockContainerCaller) LatestBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1BlockContainer.contract.Call(opts, &out, "latestBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_L1BlockContainer *L1BlockContainerCaller) LatestBlockTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1BlockContainer.contract.Call(opts, &out, "latestBlockTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Metadata is a free data retrieval call binding the contract method 0x7122ba06.
//
// Solidity: function metadata(bytes32 ) view returns(uint64 height, uint64 timestamp, uint128 baseFee)
func (_L1BlockContainer *L1BlockContainerCaller) Metadata(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Height    uint64
	Timestamp uint64
	BaseFee   *big.Int
}, error) {
	var out []interface{}
	err := _L1BlockContainer.contract.Call(opts, &out, "metadata", arg0)

	outstruct := new(struct {
		Height    uint64
		Timestamp uint64
		BaseFee   *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Height = *abi.ConvertType(out[0], new(uint64)).(*uint64)
	outstruct.Timestamp = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.BaseFee = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1BlockContainer *L1BlockContainerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1BlockContainer.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// StateRoot is a free data retrieval call binding the contract method 0x841f127c.
//
// Solidity: function stateRoot(bytes32 ) view returns(bytes32)
func (_L1BlockContainer *L1BlockContainerCaller) StateRoot(opts *bind.CallOpts, arg0 [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _L1BlockContainer.contract.Call(opts, &out, "stateRoot", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Whitelist is a free data retrieval call binding the contract method 0x93e59dc1.
//
// Solidity: function whitelist() view returns(address)
func (_L1BlockContainer *L1BlockContainerCaller) Whitelist(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1BlockContainer.contract.Call(opts, &out, "whitelist")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ImportBlockHeader is a paid mutator transaction binding the contract method 0xafbba398.
//
// Solidity: function importBlockHeader(bytes32 _blockHash, bytes _blockHeaderRLP, bool _updateGasPriceOracle) returns()
func (_L1BlockContainer *L1BlockContainerTransactor) ImportBlockHeader(opts *bind.TransactOpts, _blockHash [32]byte, _blockHeaderRLP []byte, _updateGasPriceOracle bool) (*types.Transaction, error) {
	return _L1BlockContainer.contract.Transact(opts, "importBlockHeader", _blockHash, _blockHeaderRLP, _updateGasPriceOracle)
}

// Initialize is a paid mutator transaction binding the contract method 0x9690ae23.
//
// Solidity: function initialize(bytes32 _startBlockHash, uint64 _startBlockHeight, uint64 _startBlockTimestamp, uint128 _startBlockBaseFee, bytes32 _startStateRoot) returns()
func (_L1BlockContainer *L1BlockContainerTransactor) Initialize(opts *bind.TransactOpts, _startBlockHash [32]byte, _startBlockHeight uint64, _startBlockTimestamp uint64, _startBlockBaseFee *big.Int, _startStateRoot [32]byte) (*types.Transaction, error) {
	return _L1BlockContainer.contract.Transact(opts, "initialize", _startBlockHash, _startBlockHeight, _startBlockTimestamp, _startBlockBaseFee, _startStateRoot)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1BlockContainer *L1BlockContainerTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1BlockContainer.contract.Transact(opts, "renounceOwnership")
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_L1BlockContainer *L1BlockContainerTransactor) TransferOwnership(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _L1BlockContainer.contract.Transact(opts, "transferOwnership", _newOwner)
}

// UpdateWhitelist is a paid mutator transaction binding the contract method 0x3d0f963e.
//
// Solidity: function updateWhitelist(address _newWhitelist) returns()
func (_L1BlockContainer *L1BlockContainerTransactor) UpdateWhitelist(opts *bind.TransactOpts, _newWhitelist common.Address) (*types.Transaction, error) {
	return _L1BlockContainer.contract.Transact(opts, "updateWhitelist", _newWhitelist)
}

// L1BlockContainerImportBlock represents a ImportBlock event raised by the L1BlockContainer contract.
type L1BlockContainerImportBlockEvent struct {
	BlockHash      [32]byte
	BlockHeight    *big.Int
	BlockTimestamp *big.Int
	BaseFee        *big.Int
	StateRoot      [32]byte
	raw            *types.Log // Blockchain specific contextual infos
}

// L1BlockContainerOwnershipTransferred represents a OwnershipTransferred event raised by the L1BlockContainer contract.
type L1BlockContainerOwnershipTransferredEvent struct {
	OldOwner common.Address
	NewOwner common.Address
	raw      *types.Log // Blockchain specific contextual infos
}

// L1BlockContainerUpdateWhitelist represents a UpdateWhitelist event raised by the L1BlockContainer contract.
type L1BlockContainerUpdateWhitelistEvent struct {
	OldWhitelist common.Address
	NewWhitelist common.Address
	raw          *types.Log // Blockchain specific contextual infos
}
