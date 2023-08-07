// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package predeploys

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

// L1BlockContainerMetaData contains all meta data concerning the L1BlockContainer contract.
var (
	L1BlockContainerMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blockHeight\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blockTimestamp\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"}],\"name\":\"ImportBlock\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_oldOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_oldWhitelist\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\"}],\"name\":\"UpdateWhitelist\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_blockHash\",\"type\":\"bytes32\"}],\"name\":\"getBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_blockHash\",\"type\":\"bytes32\"}],\"name\":\"getStateRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_blockHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"_blockHeaderRLP\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"_updateGasPriceOracle\",\"type\":\"bool\"}],\"name\":\"importBlockHeader\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_startBlockHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"_startBlockHeight\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_startBlockTimestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"_startBlockBaseFee\",\"type\":\"uint128\"},{\"internalType\":\"bytes32\",\"name\":\"_startStateRoot\",\"type\":\"bytes32\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"metadata\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"height\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"baseFee\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"stateRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\"}],\"name\":\"updateWhitelist\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"whitelist\",\"outputs\":[{\"internalType\":\"contractIWhitelist\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
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

// ImportBlock represents a ImportBlock event raised by the L1BlockContainer contract.
type L1BlockContainerImportBlockEvent struct {
	BlockHash      [32]byte
	BlockHeight    *big.Int
	BlockTimestamp *big.Int
	BaseFee        *big.Int
	StateRoot      [32]byte
}

// OwnershipTransferred represents a OwnershipTransferred event raised by the L1BlockContainer contract.
type L1BlockContainerOwnershipTransferredEvent struct {
	OldOwner common.Address
	NewOwner common.Address
}

// UpdateWhitelist represents a UpdateWhitelist event raised by the L1BlockContainer contract.
type L1BlockContainerUpdateWhitelistEvent struct {
	OldWhitelist common.Address
	NewWhitelist common.Address
}
