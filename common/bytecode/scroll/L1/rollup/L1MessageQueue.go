// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package rollup

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

// L1MessageQueueMetaData contains all meta data concerning the L1MessageQueue contract.
var (
	L1MessageQueueMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"startIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"count\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"skippedBitmap\",\"type\":\"uint256\"}],\"name\":\"DequeueTransaction\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"DropTransaction\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"queueIndex\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"QueueTransaction\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_oldGateway\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_newGateway\",\"type\":\"address\"}],\"name\":\"UpdateEnforcedTxGateway\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_oldGasOracle\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_newGasOracle\",\"type\":\"address\"}],\"name\":\"UpdateGasOracle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_oldMaxGasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_newMaxGasLimit\",\"type\":\"uint256\"}],\"name\":\"UpdateMaxGasLimit\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"appendCrossDomainMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"appendEnforcedTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_calldata\",\"type\":\"bytes\"}],\"name\":\"calculateIntrinsicGasFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_queueIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"computeTransactionHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"dropCrossDomainMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"enforcedTxGateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"estimateCrossDomainMessageFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gasOracle\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_queueIndex\",\"type\":\"uint256\"}],\"name\":\"getCrossDomainMessage\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_messenger\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_scrollChain\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_enforcedTxGateway\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_gasOracle\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_maxGasLimit\",\"type\":\"uint256\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxGasLimit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"messageQueue\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextCrossDomainMessageIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pendingQueueIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_startIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_count\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_skippedBitmap\",\"type\":\"uint256\"}],\"name\":\"popCrossDomainMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"scrollChain\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newGateway\",\"type\":\"address\"}],\"name\":\"updateEnforcedTxGateway\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newGasOracle\",\"type\":\"address\"}],\"name\":\"updateGasOracle\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_newMaxGasLimit\",\"type\":\"uint256\"}],\"name\":\"updateMaxGasLimit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	}
	// L1MessageQueueABI is the input ABI used to generate the binding from.
	L1MessageQueueABI *abi.ABI

	// DequeueTransaction event
	L1MessageQueueDequeueTransactionEventSignature common.Hash

	// DropTransaction event
	L1MessageQueueDropTransactionEventSignature common.Hash

	// Initialized event
	L1MessageQueueInitializedEventSignature common.Hash

	// OwnershipTransferred event
	L1MessageQueueOwnershipTransferredEventSignature common.Hash

	// QueueTransaction event
	L1MessageQueueQueueTransactionEventSignature common.Hash

	// UpdateEnforcedTxGateway event
	L1MessageQueueUpdateEnforcedTxGatewayEventSignature common.Hash

	// UpdateGasOracle event
	L1MessageQueueUpdateGasOracleEventSignature common.Hash

	// UpdateMaxGasLimit event
	L1MessageQueueUpdateMaxGasLimitEventSignature common.Hash
)

func init() {
	sigAbi, err := L1MessageQueueMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	L1MessageQueueABI = sigAbi

	// DequeueTransaction event
	L1MessageQueueDequeueTransactionEventSignature = sigAbi.Events["DequeueTransaction"].ID

	// DropTransaction event
	L1MessageQueueDropTransactionEventSignature = sigAbi.Events["DropTransaction"].ID

	// Initialized event
	L1MessageQueueInitializedEventSignature = sigAbi.Events["Initialized"].ID

	// OwnershipTransferred event
	L1MessageQueueOwnershipTransferredEventSignature = sigAbi.Events["OwnershipTransferred"].ID

	// QueueTransaction event
	L1MessageQueueQueueTransactionEventSignature = sigAbi.Events["QueueTransaction"].ID

	// UpdateEnforcedTxGateway event
	L1MessageQueueUpdateEnforcedTxGatewayEventSignature = sigAbi.Events["UpdateEnforcedTxGateway"].ID

	// UpdateGasOracle event
	L1MessageQueueUpdateGasOracleEventSignature = sigAbi.Events["UpdateGasOracle"].ID

	// UpdateMaxGasLimit event
	L1MessageQueueUpdateMaxGasLimitEventSignature = sigAbi.Events["UpdateMaxGasLimit"].ID

}

// DequeueTransaction represents a DequeueTransaction event raised by the L1MessageQueue contract.
type L1MessageQueueDequeueTransactionEvent struct {
	StartIndex    *big.Int
	Count         *big.Int
	SkippedBitmap *big.Int
}

// DropTransaction represents a DropTransaction event raised by the L1MessageQueue contract.
type L1MessageQueueDropTransactionEvent struct {
	Index *big.Int
}

// Initialized represents a Initialized event raised by the L1MessageQueue contract.
type L1MessageQueueInitializedEvent struct {
	Version uint8
}

// OwnershipTransferred represents a OwnershipTransferred event raised by the L1MessageQueue contract.
type L1MessageQueueOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
}

// QueueTransaction represents a QueueTransaction event raised by the L1MessageQueue contract.
type L1MessageQueueQueueTransactionEvent struct {
	Sender     common.Address
	Target     common.Address
	Value      *big.Int
	QueueIndex uint64
	GasLimit   *big.Int
	Data       []byte
}

// UpdateEnforcedTxGateway represents a UpdateEnforcedTxGateway event raised by the L1MessageQueue contract.
type L1MessageQueueUpdateEnforcedTxGatewayEvent struct {
	OldGateway common.Address
	NewGateway common.Address
}

// UpdateGasOracle represents a UpdateGasOracle event raised by the L1MessageQueue contract.
type L1MessageQueueUpdateGasOracleEvent struct {
	OldGasOracle common.Address
	NewGasOracle common.Address
}

// UpdateMaxGasLimit represents a UpdateMaxGasLimit event raised by the L1MessageQueue contract.
type L1MessageQueueUpdateMaxGasLimitEvent struct {
	OldMaxGasLimit *big.Int
	NewMaxGasLimit *big.Int
}
