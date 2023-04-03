package config

import (
	"github.com/scroll-tech/go-ethereum/rpc"

	"github.com/scroll-tech/go-ethereum/common"

	"scroll-tech/common/types"
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
	// The batch_proposer config
	BatchProposerConfig *BatchProposerConfig `json:"batch_proposer_config"`
}

// BatchProposerConfig loads l2watcher batch_proposer configuration items.
type BatchProposerConfig struct {
	// Proof generation frequency, generating proof every k blocks
	ProofGenerationFreq uint64 `json:"proof_generation_freq"`
	// Txnum threshold in a batch
	BatchTxNumThreshold uint64 `json:"batch_tx_num_threshold"`
	// Gas threshold in a batch
	BatchGasThreshold uint64 `json:"batch_gas_threshold"`
	// Time waited to generate a batch even if gas_threshold not met
	BatchTimeSec uint64 `json:"batch_time_sec"`
	// Time waited to commit batches before the calldata met CommitTxCalldataSizeLimit
	BatchCommitTimeSec uint64 `json:"batch_commit_time_sec"`
	// Max number of blocks in a batch
	BatchBlocksLimit uint64 `json:"batch_blocks_limit"`
	// Commit tx calldata size limit in bytes, target to cap the gas use of commit tx at 2M gas
	CommitTxCalldataSizeLimit uint64 `json:"commit_tx_calldata_size_limit"`
	// Commit tx calldata min size limit in bytes
	CommitTxCalldataMinSize uint64 `json:"commit_tx_calldata_min_size,omitempty"`
	// The public input hash config
	PublicInputConfig *types.PublicInputHashConfig `json:"public_input_config"`
}
