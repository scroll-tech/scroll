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
}

// ChunkProposerConfig loads chunk_proposer configuration items.
type ChunkProposerConfig struct {
	MaxGasPerChunk         uint64 `json:"max_gas_per_chunk"`
	MaxTxNumPerChunk       uint64 `json:"max_tx_num_chunk"`
	MaxPayloadSizePerChunk uint64 `json:"max_payload_size_per_chunk"`
	MinPayloadSizePerChunk uint64 `json:"min_payload_size_per_chunk"`
}
