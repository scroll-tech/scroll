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

// ScrollChainMetaData contains all meta data concerning the ScrollChain contract.
var (
	ScrollChainMetaData = &bind.MetaData{
		ABI: "[{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_chainId\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"batchHash\",\"type\":\"bytes32\"}],\"name\":\"CommitBatch\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"batchHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"withdrawRoot\",\"type\":\"bytes32\"}],\"name\":\"FinalizeBatch\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"batchHash\",\"type\":\"bytes32\"}],\"name\":\"RevertBatch\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"oldMaxNumL2TxInChunk\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newMaxNumL2TxInChunk\",\"type\":\"uint256\"}],\"name\":\"UpdateMaxNumL2TxInChunk\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"status\",\"type\":\"bool\"}],\"name\":\"UpdateProver\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"status\",\"type\":\"bool\"}],\"name\":\"UpdateSequencer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldVerifier\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newVerifier\",\"type\":\"address\"}],\"name\":\"UpdateVerifier\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"}],\"name\":\"addProver\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"}],\"name\":\"addSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"_version\",\"type\":\"uint8\"},{\"internalType\":\"bytes\",\"name\":\"_parentBatchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes[]\",\"name\":\"_chunks\",\"type\":\"bytes[]\"},{\"internalType\":\"bytes\",\"name\":\"_skippedL1MessageBitmap\",\"type\":\"bytes\"}],\"name\":\"commitBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"committedBatches\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"_prevStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_postStateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_withdrawRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"_aggrProof\",\"type\":\"bytes\"}],\"name\":\"finalizeBatchWithProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"finalizedStateRoots\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"_stateRoot\",\"type\":\"bytes32\"}],\"name\":\"importGenesisBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_messageQueue\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_verifier\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_maxNumL2TxInChunk\",\"type\":\"uint256\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"}],\"name\":\"isBatchFinalized\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isProver\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isSequencer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"lastFinalizedBatchIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"layer2ChainId\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxNumL2TxInChunk\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messageQueue\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"}],\"name\":\"removeProver\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"}],\"name\":\"removeSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_batchHeader\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_count\",\"type\":\"uint256\"}],\"name\":\"revertBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"_status\",\"type\":\"bool\"}],\"name\":\"setPause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_maxNumL2TxInChunk\",\"type\":\"uint256\"}],\"name\":\"updateMaxNumL2TxInChunk\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newVerifier\",\"type\":\"address\"}],\"name\":\"updateVerifier\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"verifier\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"withdrawRoots\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
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

// CommitBatch represents a CommitBatch event raised by the ScrollChain contract.
type ScrollChainCommitBatchEvent struct {
	BatchIndex *big.Int
	BatchHash  [32]byte
}

// FinalizeBatch represents a FinalizeBatch event raised by the ScrollChain contract.
type ScrollChainFinalizeBatchEvent struct {
	BatchIndex   *big.Int
	BatchHash    [32]byte
	StateRoot    [32]byte
	WithdrawRoot [32]byte
}

// Initialized represents a Initialized event raised by the ScrollChain contract.
type ScrollChainInitializedEvent struct {
	Version uint8
}

// OwnershipTransferred represents a OwnershipTransferred event raised by the ScrollChain contract.
type ScrollChainOwnershipTransferredEvent struct {
	PreviousOwner common.Address
	NewOwner      common.Address
}

// Paused represents a Paused event raised by the ScrollChain contract.
type ScrollChainPausedEvent struct {
	Account common.Address
}

// RevertBatch represents a RevertBatch event raised by the ScrollChain contract.
type ScrollChainRevertBatchEvent struct {
	BatchIndex *big.Int
	BatchHash  [32]byte
}

// Unpaused represents a Unpaused event raised by the ScrollChain contract.
type ScrollChainUnpausedEvent struct {
	Account common.Address
}

// UpdateMaxNumL2TxInChunk represents a UpdateMaxNumL2TxInChunk event raised by the ScrollChain contract.
type ScrollChainUpdateMaxNumL2TxInChunkEvent struct {
	OldMaxNumL2TxInChunk *big.Int
	NewMaxNumL2TxInChunk *big.Int
}

// UpdateProver represents a UpdateProver event raised by the ScrollChain contract.
type ScrollChainUpdateProverEvent struct {
	Account common.Address
	Status  bool
}

// UpdateSequencer represents a UpdateSequencer event raised by the ScrollChain contract.
type ScrollChainUpdateSequencerEvent struct {
	Account common.Address
	Status  bool
}

// UpdateVerifier represents a UpdateVerifier event raised by the ScrollChain contract.
type ScrollChainUpdateVerifierEvent struct {
	OldVerifier common.Address
	NewVerifier common.Address
}
