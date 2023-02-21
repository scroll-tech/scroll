package config

import (
	"encoding/json"

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
	// Max number of blocks in a batch
	BatchBlocksLimit uint64 `json:"batch_blocks_limit"`
	// Commit tx calldata size limit, target to cap the gas use of commit tx at 2M gas
	CommitTxCalldataSizeLimit uint64 `json:"commit_tx_calldata_size_limit"`
	// Skip generating proof when that opcodes appeared
	SkippedOpcodes map[string]struct{} `json:"-"`
	// The public input hash config
	PublicInputConfig *types.PublicInputHashConfig `json:"public_input_config"`
}

// batchProposerConfigAlias RelayerConfig alias name
type batchProposerConfigAlias BatchProposerConfig

// UnmarshalJSON unmarshal BatchProposerConfig config struct.
func (b *BatchProposerConfig) UnmarshalJSON(input []byte) error {
	var jsonConfig struct {
		batchProposerConfigAlias
		SkippedOpcodes []string `json:"skipped_opcodes,omitempty"`
	}
	if err := json.Unmarshal(input, &jsonConfig); err != nil {
		return err
	}

	*b = BatchProposerConfig(jsonConfig.batchProposerConfigAlias)
	b.SkippedOpcodes = make(map[string]struct{}, len(jsonConfig.SkippedOpcodes))
	for _, opcode := range jsonConfig.SkippedOpcodes {
		b.SkippedOpcodes[opcode] = struct{}{}
	}
	return nil
}

// MarshalJSON marshal BatchProposerConfig in order to transfer skipOpcodes.
func (b *BatchProposerConfig) MarshalJSON() ([]byte, error) {
	jsonConfig := struct {
		batchProposerConfigAlias
		SkippedOpcodes []string `json:"skipped_opcodes,omitempty"`
	}{batchProposerConfigAlias(*b), nil}

	// Load skipOpcodes.
	for op := range b.SkippedOpcodes {
		jsonConfig.SkippedOpcodes = append(jsonConfig.SkippedOpcodes, op)
	}

	return json.Marshal(&jsonConfig)
}
