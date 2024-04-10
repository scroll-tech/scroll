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
	MaxBlockNumPerChunk             uint64  `json:"max_block_num_per_chunk"`
	MaxTxNumPerChunk                uint64  `json:"max_tx_num_per_chunk"`
	MaxL1CommitGasPerChunk          uint64  `json:"max_l1_commit_gas_per_chunk"`
	MaxL1CommitCalldataSizePerChunk uint64  `json:"max_l1_commit_calldata_size_per_chunk"`
	ChunkTimeoutSec                 uint64  `json:"chunk_timeout_sec"`
	MaxRowConsumptionPerChunk       uint64  `json:"max_row_consumption_per_chunk"`
	GasCostIncreaseMultiplier       float64 `json:"gas_cost_increase_multiplier"`
}

// BatchProposerConfig loads batch_proposer configuration items.
type BatchProposerConfig struct {
	MaxChunkNumPerBatch             uint64  `json:"max_chunk_num_per_batch"`
	MaxL1CommitGasPerBatch          uint64  `json:"max_l1_commit_gas_per_batch"`
	MaxL1CommitCalldataSizePerBatch uint64  `json:"max_l1_commit_calldata_size_per_batch"`
	BatchTimeoutSec                 uint64  `json:"batch_timeout_sec"`
	GasCostIncreaseMultiplier       float64 `json:"gas_cost_increase_multiplier"`
}
