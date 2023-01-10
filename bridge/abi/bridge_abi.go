package bridgeabi

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
)

var (
	// RollupMetaABI holds information about ZKRollup contracts' context and available invokable methods.
	RollupABI *abi.ABI
	// L1MessengerMetaABI holds information about L1ScrollMessenger contract's context and available invokable methods.
	L1MessengerABI *abi.ABI
	// L1MessageQueueABI holds information about L1MessageQueue contract's context and available invokable methods.
	L1MessageQueueABI *abi.ABI

	// L2MessengerMetaABI holds information about L2ScrollMessenger contract's context and available invokable methods.
	L2MessengerABI *abi.ABI
	// L1BlockContainerMetaABI holds information about L1BlockContainer contract's context and available invokable methods.
	L1BlockContainerABI *abi.ABI
	// L2MessageQueueABI holds information about L2MessageQueue contract's context and available invokable methods.
	L2MessageQueueABI *abi.ABI

	// L1SentMessageEventSignature = keccak256("SentMessage(address,address,uint256,uint256,uint256,bytes,uint256,uint256)")
	L1SentMessageEventSignature common.Hash
	// L1RelayedMessageEventSignature = keccak256("RelayedMessage(bytes32)")
	L1RelayedMessageEventSignature common.Hash
	// L1FailedRelayedMessageEventSignature = keccak256("FailedRelayedMessage(bytes32)")
	L1FailedRelayedMessageEventSignature common.Hash

	// L1CommitBatchEventSignature = keccak256("CommitBatch(bytes32,bytes32,uint256,bytes32)")
	L1CommitBatchEventSignature common.Hash
	// L1FinalizeBatchEventSignature = keccak256("FinalizeBatch(bytes32,bytes32,uint256,bytes32)")
	L1FinalizeBatchEventSignature common.Hash

	// L1AppendMessageEventSignature = keccak256("AppendMessage(bytes32)")
	L1AppendMessageEventSignature common.Hash

	// L2SentMessageEventSignature = keccak256("SentMessage(address,address,uint256,uint256,uint256,bytes,uint256,uint256)")
	L2SentMessageEventSignature common.Hash
	// L2RelayedMessageEventSignature = keccak256("RelayedMessage(bytes32)")
	L2RelayedMessageEventSignature common.Hash
	// L2FailedRelayedMessageEventSignature = keccak256("FailedRelayedMessage(bytes32)")
	L2FailedRelayedMessageEventSignature common.Hash

	// L2ImportBlockEventSignature = keccak256("ImportBlock(bytes32,uint256,uint256,bytes32)")
	L2ImportBlockEventSignature common.Hash

	// L2AppendMessageEventSignature = keccak256("AppendMessage(uint256,bytes32)")
	L2AppendMessageEventSignature common.Hash
)

func init() {
	L1MessengerABI, _ = L1MessengerMetaData.GetAbi()
	RollupABI, _ = RollupMetaData.GetAbi()
	L1MessageQueueABI, _ = L1MessageQueueMetaData.GetAbi()

	L2MessengerABI, _ = L2MessengerMetaData.GetAbi()
	L1BlockContainerABI, _ = L1BlockContainerMetaData.GetAbi()
	L2MessageQueueABI, _ = L2MessageQueueMetaData.GetAbi()

	L1SentMessageEventSignature = L1MessengerABI.Events["SentMessage"].ID
	L1RelayedMessageEventSignature = L1MessengerABI.Events["RelayedMessage"].ID
	L1FailedRelayedMessageEventSignature = L1MessengerABI.Events["FailedRelayedMessage"].ID

	L1CommitBatchEventSignature = RollupABI.Events["CommitBatch"].ID
	L1FinalizeBatchEventSignature = RollupABI.Events["FinalizeBatch"].ID

	L1AppendMessageEventSignature = L1MessageQueueABI.Events["AppendMessage"].ID

	L2SentMessageEventSignature = L2MessengerABI.Events["SentMessage"].ID
	L2RelayedMessageEventSignature = L2MessengerABI.Events["RelayedMessage"].ID
	L2FailedRelayedMessageEventSignature = L2MessengerABI.Events["FailedRelayedMessage"].ID

	L2ImportBlockEventSignature = L1BlockContainerABI.Events["ImportBlock"].ID

	L2AppendMessageEventSignature = L2MessageQueueABI.Events["AppendMessage"].ID
}

// Generated manually from abigen and only necessary events and mutable calls are kept.

// L1MessengerMetaData contains all meta data concerning the L1Messenger contract.
var L1MessengerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"FailedRelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"MessageDropped\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"RelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"}],\"name\":\"SentMessage\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"dropMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"messageRootProof\",\"type\":\"bytes32[]\"}],\"internalType\":\"structIL1ScrollMessenger.L2MessageProof\",\"name\":\"_proof\",\"type\":\"tuple\"}],\"name\":\"relayMessageWithProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_queueIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_oldGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"_newGasLimit\",\"type\":\"uint32\"}],\"name\":\"replayMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"xDomainMessageSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// RollupMetaData contains all meta data concerning the Rollup contract.
var RollupMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_batchId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_parentHash\",\"type\":\"bytes32\"}],\"name\":\"CommitBatch\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_batchId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_parentHash\",\"type\":\"bytes32\"}],\"name\":\"FinalizeBatch\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_batchId\",\"type\":\"bytes32\"}],\"name\":\"RevertBatch\",\"type\":\"event\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint64\",\"name\":\"batchIndex\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"parentHash\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"parentHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"blockHeight\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"gasUsed\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"nonce\",\"type\":\"uint64\"},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"gas\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"gasPrice\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"r\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"s\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"v\",\"type\":\"uint64\"}],\"internalType\":\"structIZKRollup.Layer2Transaction[]\",\"name\":\"txs\",\"type\":\"tuple[]\"},{\"internalType\":\"bytes32\",\"name\":\"messageRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structIZKRollup.Layer2BlockHeader[]\",\"name\":\"blocks\",\"type\":\"tuple[]\"}],\"internalType\":\"structIZKRollup.Layer2Batch\",\"name\":\"_batch\",\"type\":\"tuple\"}],\"name\":\"commitBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_batchId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256[]\",\"name\":\"_proof\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"_instances\",\"type\":\"uint256[]\"}],\"name\":\"finalizeBatchWithProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"name\":\"getL2MessageRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"name\":\"isBlockFinalized\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockHeight\",\"type\":\"uint256\"}],\"name\":\"isBlockFinalized\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"layer2GasLimit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_batchId\",\"type\":\"bytes32\"}],\"name\":\"revertBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// L1MessageQueueMetaData contains all meta data concerning the L1MessageQueue contract.
var L1MessageQueueMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"AppendMessage\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_msgHash\",\"type\":\"bytes32\"}],\"name\":\"appendMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_msgHash\",\"type\":\"bytes32\"}],\"name\":\"hasMessage\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextMessageIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Sigs: map[string]string{
		"600a2e77": "appendMessage(bytes32)",
		"e90cc719": "hasMessage(bytes32)",
		"26aad7b7": "nextMessageIndex()",
	},
}

// L2MessengerMetaData contains all meta data concerning the L2Messenger contract.
var L2MessengerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"FailedRelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"MessageDropped\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"RelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"}],\"name\":\"SentMessage\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"dropMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"stateRootProof\",\"type\":\"bytes\"}],\"internalType\":\"structIL2ScrollMessenger.L1MessageProof\",\"name\":\"_proof\",\"type\":\"tuple\"}],\"name\":\"relayMessageWithProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"xDomainMessageSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// L1BlockContainerMetaData contains all meta data concerning the L1BlockContainer contract.
var L1BlockContainerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blockHeight\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"blockTimestamp\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"}],\"name\":\"ImportBlock\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"name\":\"getBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"name\":\"getStateRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"blockHeaderRLP\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"importBlockHeader\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"verifyMessageExecutionStatus\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"executed\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"verifyMessageInclusionStatus\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"included\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// L2MessageQueueMetaData contains all meta data concerning the L2MessageQueue contract.
var L2MessageQueueMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_messenger\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"AppendMessage\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_messageHash\",\"type\":\"bytes32\"}],\"name\":\"appendMessage\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"branches\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messageRoot\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextMessageIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"sentMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// IL1ScrollMessengerL2MessageProof is an auto generated low-level Go binding around an user-defined struct.
type IL1ScrollMessengerL2MessageProof struct {
	BlockHash        common.Hash
	MessageRootProof []common.Hash
}

// IL2ScrollMessengerL1MessageProof is an auto generated low-level Go binding around an user-defined struct.
type IL2ScrollMessengerL1MessageProof struct {
	BlockHash      common.Hash
	StateRootProof []byte
}

// IZKRollupLayer2Batch is an auto generated low-level Go binding around an user-defined struct.
type IZKRollupLayer2Batch struct {
	BatchIndex uint64
	ParentHash [32]byte
	Blocks     []IZKRollupLayer2BlockHeader
}

// IZKRollupLayer2BlockHeader is an auto generated low-level Go binding around an user-defined struct.
type IZKRollupLayer2BlockHeader struct {
	BlockHash   [32]byte
	ParentHash  [32]byte
	BaseFee     *big.Int
	StateRoot   [32]byte
	BlockHeight uint64
	GasUsed     uint64
	Timestamp   uint64
	ExtraData   []byte
	Txs         []IZKRollupLayer2Transaction
	MessageRoot common.Hash
}

// IZKRollupLayer2Transaction is an auto generated low-level Go binding around an user-defined struct.
type IZKRollupLayer2Transaction struct {
	Nonce    uint64
	Target   common.Address
	Gas      uint64
	GasPrice *big.Int
	Value    *big.Int
	Data     []byte
	R        *big.Int
	S        *big.Int
	V        uint64
}

// L1AppendMessageEvent represents a AppendMessage event raised by the L1MessageQueue contract.
type L1AppendMessageEvent struct {
	MsgHash common.Hash
}

// L1CommitBatchEvent represents a CommitBatch event raised by the ZKRollup contract.
type L1CommitBatchEvent struct {
	BatchId    common.Hash
	BatchHash  common.Hash
	BatchIndex *big.Int
	ParentHash common.Hash
}

// L1FinalizeBatchEvent represents a FinalizeBatch event raised by the ZKRollup contract.
type L1FinalizeBatchEvent struct {
	BatchId    common.Hash
	BatchHash  common.Hash
	BatchIndex *big.Int
	ParentHash common.Hash
}

// L1RevertBatchEvent represents a RevertBatch event raised by the ZKRollup contract.
type L1RevertBatchEvent struct {
	BatchId common.Hash
}

// L1SentMessageEvent represents a SentMessage event raised by the L1ScrollMessenger contract.
type L1SentMessageEvent struct {
	Target       common.Address
	Sender       common.Address
	Value        *big.Int
	Fee          *big.Int
	Deadline     *big.Int
	Message      []byte
	MessageNonce *big.Int
	GasLimit     *big.Int
}

// L1FailedRelayedMessageEvent represents a FailedRelayedMessage event raised by the L1ScrollMessenger contract.
type L1FailedRelayedMessageEvent struct {
	MsgHash common.Hash
}

// L1MessageDroppedEvent represents a MessageDropped event raised by the L1ScrollMessenger contract.
type L1MessageDroppedEvent struct {
	MsgHash common.Hash
}

// L1RelayedMessageEvent represents a RelayedMessage event raised by the L1ScrollMessenger contract.
type L1RelayedMessageEvent struct {
	MsgHash common.Hash
}

// L2AppendMessageEvent represents a AppendMessage event raised by the L2MessageQueue contract.
type L2AppendMessageEvent struct {
	Index       *big.Int
	MessageHash common.Hash
}

// L2ImportBlockEvent represents a ImportBlock event raised by the L1BlockContainer contract.
type L2ImportBlockEvent struct {
	BlockHash      common.Hash
	BlockHeight    *big.Int
	BlockTimestamp *big.Int
	StateRoot      common.Hash
}

// L2SentMessageEvent represents a SentMessage event raised by the L2ScrollMessenger contract.
type L2SentMessageEvent struct {
	Target       common.Address
	Sender       common.Address
	Value        *big.Int
	Fee          *big.Int
	Deadline     *big.Int
	Message      []byte
	MessageNonce *big.Int
	GasLimit     *big.Int
}

// L2FailedRelayedMessageEvent represents a FailedRelayedMessage event raised by the L2ScrollMessenger contract.
type L2FailedRelayedMessageEvent struct {
	MsgHash common.Hash
}

// L2MessageDroppedEvent represents a MessageDropped event raised by the L2ScrollMessenger contract.
type L2MessageDroppedEvent struct {
	MsgHash common.Hash
}

// L2RelayedMessageEvent represents a RelayedMessage event raised by the L2ScrollMessenger contract.
type L2RelayedMessageEvent struct {
	MsgHash common.Hash
}
