package config

import (
	"github.com/scroll-tech/go-ethereum/rpc"

	"github.com/scroll-tech/go-ethereum/common"
)

// L2Config loads l2geth configuration items.
type L2Config struct {
	// Confirmations block height confirmations number.
	Confirmations rpc.BlockNumber `json:"confirmations"`
	// l2geth node url.
	Endpoint string `json:"endpoint"`
	// The messenger contract address deployed on layer 2 chain.
	L2MessengerAddress common.Address `json:"l2_messenger_address"`
	// The L2MessageQueue contract address deployed on layer 2 chain.
	L2MessageQueueAddress common.Address `json:"l2_message_queue_address"`
	// The WithdrawTrieRootSlot in L2MessageQueue contract.
	WithdrawTrieRootSlot common.Hash `json:"withdraw_trie_root_slot,omitempty"`
	// The relayer config
	RelayerConfig *RelayerConfig `json:"relayer_config"`
	// The chunk_proposer config
	ChunkProposerConfig *ChunkProposerConfig `json:"chunk_proposer_config"`
	// The batch_proposer config
	BatchProposerConfig *BatchProposerConfig `json:"batch_proposer_config"`
}

// ChunkProposerConfig loads chunk_proposer configuration items.
type ChunkProposerConfig struct {
	ChunkTimeoutSec         uint64 `json:"chunk_timeout_sec"`
	MaxGasPerChunk          uint64 `json:"max_gas_per_chunk"`
	MaxL2TxNumPerChunk      uint64 `json:"max_l2_tx_num_per_chunk"` // TODO: could this be replaced with MaxCalldataGasPerChunk?
	MaxCalldataGasPerChunk  uint64 `json:"max_calldata_gas_per_chunk"`
	MaxCalldataSizePerChunk uint64 `json:"max_calldata_size_per_chunk"`
	MinCalldataSizePerChunk uint64 `json:"min_calldata_size_per_chunk"`
}

// BatchProposerConfig loads batch_proposer configuration items.
type BatchProposerConfig struct {
	BatchTimeoutSec         uint64 `json:"batch_timeout_sec"`
	MaxCalldataGasPerBatch  uint64 `json:"max_calldata_gas_per_batch"`
	MaxCalldataSizePerBatch uint64 `json:"max_calldata_size_per_batch"`
	MinCalldataSizePerBatch uint64 `json:"min_calldata_size_per_batch"`
}
