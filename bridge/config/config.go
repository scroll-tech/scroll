package config

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"

	"scroll-tech/common/utils"

	db_config "scroll-tech/database"
)

// SenderConfig The config for transaction sender
type SenderConfig struct {
	// The RPC endpoint of the ethereum or scroll public node.
	Endpoint string `json:"endpoint"`
	// The time to trigger check pending txs in sender.
	CheckPendingTime uint64 `json:"check_pending_time"`
	// The number of blocks to wait to escalate increase gas price of the transaction.
	EscalateBlocks uint64 `json:"escalate_blocks"`
	// The gap number between a block be confirmed and the latest block.
	Confirmations uint64 `json:"confirmations"`
	// The numerator of gas price escalate multiple.
	EscalateMultipleNum uint64 `json:"escalate_multiple_num"`
	// The denominator of gas price escalate multiple.
	EscalateMultipleDen uint64 `json:"escalate_multiple_den"`
	// The maximum gas price can be used to send transaction.
	MaxGasPrice uint64 `json:"max_gas_price"`
	// The transaction type to use: LegacyTx, AccessListTx, DynamicFeeTx
	TxType string `json:"tx_type"`
	// The min balance set for check and set balance for sender's accounts.
	MinBalance *big.Int `json:"min_balance,omitempty"`
}

// L1Config loads l1eth configuration items.
type L1Config struct {
	// Confirmations block height confirmations number.
	Confirmations uint64 `json:"confirmations"`
	// l1 chainID.
	ChainID int64 `json:"chain_id"`
	// l1 eth node url.
	Endpoint string `json:"endpoint"`
	// The start height to sync event from layer 1
	StartHeight uint64 `json:"start_height"`
	// The messenger contract address deployed on layer 1 chain.
	L1MessengerAddress common.Address `json:"l1_messenger_address,omitempty"`
	// The relayer config
	RelayerConfig *RelayerConfig `json:"relayer_config"`
}

// L2Config loads l2geth configuration items.
type L2Config struct {
	// Confirmations block height confirmations number.
	Confirmations uint64 `json:"confirmations"`
	// l2geth chainId.
	ChainID int64 `json:"chain_id"`
	// l2geth node url.
	Endpoint string `json:"endpoint"`
	// The messenger contract address deployed on layer 2 chain.
	L2MessengerAddress common.Address `json:"l2_messenger_address,omitempty"`
	// The relayer config
	RelayerConfig *RelayerConfig `json:"relayer_config"`
	// The batch_proposer config
	BatchProposerConfig *BatchProposerConfig `json:"batch_proposer_config"`
}

// BatchProposerConfig loads l2watcher batch_proposer configuration items.
type BatchProposerConfig struct {
	// Proof generation frequency, generating proof every k blocks
	ProofGenerationFreq uint64 `json:"proof_generation_freq"`
	// Skip generating proof when that opcodes appeared
	SkippedOpcodes map[string]struct{} `json:"-"`
	// Gas threshold in a batch
	BatchGasThreshold uint64 `json:"batch_gas_threshold"`
	// Time waited to generate a batch even if gas_threshold not met
	BatchTimeSec uint64 `json:"batch_time_sec"`
	// Max number of blocks in a batch
	BatchBlocksLimit uint64 `json:"batch_block_limit"`
}

// // L2ConfigAlias L2Config alias name, designed just for unmarshal.
// type L2ConfigAlias L2Config

// // UnmarshalJSON unmarshal l2config.
// func (l2 *L2Config) UnmarshalJSON(input []byte) error {
// 	var jsonConfig struct {
// 		L2ConfigAlias
// 		SkippedOpcodes []string `json:"skipped_opcodes"`
// 	}
// 	if err := json.Unmarshal(input, &jsonConfig); err != nil {
// 		return err
// 	}
// 	*l2 = L2Config(jsonConfig.L2ConfigAlias)
// 	l2.SkippedOpcodes = make(map[string]struct{}, len(jsonConfig.SkippedOpcodes))
// 	for _, opcode := range jsonConfig.SkippedOpcodes {
// 		l2.SkippedOpcodes[opcode] = struct{}{}
// 	}
// 	if 0 == l2.ProofGenerationFreq {
// 		l2.ProofGenerationFreq = 1
// 	}
// 	return nil
// }

// RelayerConfig loads relayer configuration items.
// What we need to pay attention to is that
// `MessageSenderPrivateKeys` and `RollupSenderPrivateKeys` cannot have common private keys.
type RelayerConfig struct {
	// RollupContractAddress store the rollup contract address.
	RollupContractAddress common.Address `json:"rollup_contract_address,omitempty"`
	// MessengerContractAddress store the scroll messenger contract address.
	MessengerContractAddress common.Address `json:"messenger_contract_address"`
	// sender config
	SenderConfig *SenderConfig `json:"sender_config"`
	// The private key of the relayer
	MessageSenderPrivateKeys []*ecdsa.PrivateKey `json:"-"`
	RollupSenderPrivateKeys  []*ecdsa.PrivateKey `json:"-"`
}

// RelayerConfigAlias RelayerConfig alias name
type RelayerConfigAlias RelayerConfig

// UnmarshalJSON unmarshal relayer_config struct.
func (r *RelayerConfig) UnmarshalJSON(input []byte) error {
	var jsonConfig struct {
		RelayerConfigAlias
		// The private key of the relayer
		MessageSenderPrivateKeys []string `json:"message_sender_private_keys"`
		RollupSenderPrivateKeys  []string `json:"roller_sender_private_keys,omitempty"`
	}
	if err := json.Unmarshal(input, &jsonConfig); err != nil {
		return err
	}

	// Get messenger private key list.
	*r = RelayerConfig(jsonConfig.RelayerConfigAlias)
	for _, privStr := range jsonConfig.MessageSenderPrivateKeys {
		priv, err := crypto.ToECDSA(common.FromHex(privStr))
		if err != nil {
			return fmt.Errorf("incorrect private_key_list format, err: %v", err)
		}
		r.MessageSenderPrivateKeys = append(r.MessageSenderPrivateKeys, priv)
	}

	// Get rollup private key
	for _, privStr := range jsonConfig.RollupSenderPrivateKeys {
		priv, err := crypto.ToECDSA(common.FromHex(privStr))
		if err != nil {
			return fmt.Errorf("incorrect roller_private_key format, err: %v", err)
		}
		r.RollupSenderPrivateKeys = append(r.RollupSenderPrivateKeys, priv)
	}

	return nil
}

// Config load configuration items.
type Config struct {
	L1Config *L1Config           `json:"l1_config"`
	L2Config *L2Config           `json:"l2_config"`
	DBConfig *db_config.DBConfig `json:"db_config"`
}

// NewConfig returns a new instance of Config.
func NewConfig(file string) (*Config, error) {
	buf, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = json.Unmarshal(buf, cfg)
	if err != nil {
		return nil, err
	}

	// cover value by env fields
	cfg.DBConfig.DSN = utils.GetEnvWithDefault("DB_DSN", cfg.DBConfig.DSN)
	cfg.DBConfig.DriverName = utils.GetEnvWithDefault("DB_DRIVER", cfg.DBConfig.DriverName)

	return cfg, nil
}
