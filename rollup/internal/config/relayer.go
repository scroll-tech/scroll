package config

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/rpc"
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
	Confirmations rpc.BlockNumber `json:"confirmations"`
	// The numerator of gas price escalate multiple.
	EscalateMultipleNum uint64 `json:"escalate_multiple_num"`
	// The denominator of gas price escalate multiple.
	EscalateMultipleDen uint64 `json:"escalate_multiple_den"`
	// The maximum gas price can be used to send transaction.
	MaxGasPrice uint64 `json:"max_gas_price"`
	// The minimum gas tip can be used to send transaction.
	MinGasTip uint64 `json:"min_gas_tip"`
	// The maximum blob gas price can be used to send transaction.
	MaxBlobGasPrice uint64 `json:"max_blob_gas_price"`
	// The transaction type to use: LegacyTx, DynamicFeeTx, BlobTx
	TxType string `json:"tx_type"`
}

// ChainMonitor this config is used to get batch status from chain_monitor API.
type ChainMonitor struct {
	Enabled  bool   `json:"enabled"`
	TimeOut  int    `json:"timeout"`
	TryTimes int    `json:"try_times"`
	BaseURL  string `json:"base_url"`
}

// RelayerConfig loads relayer configuration items.
// What we need to pay attention to is that
type RelayerConfig struct {
	// RollupContractAddress store the rollup contract address.
	RollupContractAddress common.Address `json:"rollup_contract_address,omitempty"`
	// GasPriceOracleContractAddress store the scroll messenger contract address.
	GasPriceOracleContractAddress common.Address `json:"gas_price_oracle_contract_address"`
	// sender config
	SenderConfig *SenderConfig `json:"sender_config"`
	// gas oracle config
	GasOracleConfig *GasOracleConfig `json:"gas_oracle_config"`
	// ChainMonitor config of monitoring service
	ChainMonitor *ChainMonitor `json:"chain_monitor"`
	// L1CommitGasLimitMultiplier multiplier for fallback gas limit in commitBatch txs
	L1CommitGasLimitMultiplier float64 `json:"l1_commit_gas_limit_multiplier,omitempty"`
	// The private key of the relayer
	GasOracleSenderPrivateKey *ecdsa.PrivateKey `json:"-"`
	CommitSenderPrivateKey    *ecdsa.PrivateKey `json:"-"`
	FinalizeSenderPrivateKey  *ecdsa.PrivateKey `json:"-"`

	// Indicates if bypass features specific to testing environments are enabled.
	EnableTestEnvBypassFeatures bool `json:"enable_test_env_bypass_features"`
	// The timeout in seconds for finalizing a batch without proof, only used when EnableTestEnvBypassFeatures is true.
	FinalizeBatchWithoutProofTimeoutSec uint64 `json:"finalize_batch_without_proof_timeout_sec"`
}

// GasOracleConfig The config for updating gas price oracle.
type GasOracleConfig struct {
	// MinGasPrice store the minimum gas price to set.
	MinGasPrice uint64 `json:"min_gas_price"`
	// GasPriceDiff is the minimum percentage of gas price difference to update gas oracle.
	GasPriceDiff uint64 `json:"gas_price_diff"`

	// The following configs are only for updating L1 gas price, used for sender in L2.
	// The weight for L1 base fee.
	L1BaseFeeWeight float64 `json:"l1_base_fee_weight"`
	// The weight for L1 blob base fee.
	L1BlobBaseFeeWeight float64 `json:"l1_blob_base_fee_weight"`
}

// relayerConfigAlias RelayerConfig alias name
type relayerConfigAlias RelayerConfig

func convertAndCheck(key string, uniqueAddressesSet map[string]struct{}) (*ecdsa.PrivateKey, error) {
	if key == "" {
		return nil, nil
	}

	privKey, err := crypto.ToECDSA(common.FromHex(key))
	if err != nil {
		return nil, err
	}

	addr := crypto.PubkeyToAddress(privKey.PublicKey).Hex()
	if _, exists := uniqueAddressesSet[addr]; exists {
		return nil, fmt.Errorf("detected duplicated address for private key: %s", addr)
	}
	uniqueAddressesSet[addr] = struct{}{}

	return privKey, nil
}

// UnmarshalJSON unmarshal relayer_config struct.
func (r *RelayerConfig) UnmarshalJSON(input []byte) error {
	var privateKeysConfig struct {
		relayerConfigAlias
		GasOracleSenderPrivateKey string `json:"gas_oracle_sender_private_key"`
		CommitSenderPrivateKey    string `json:"commit_sender_private_key"`
		FinalizeSenderPrivateKey  string `json:"finalize_sender_private_key"`
	}
	var err error
	if err = json.Unmarshal(input, &privateKeysConfig); err != nil {
		return fmt.Errorf("failed to unmarshal private keys config: %w", err)
	}

	*r = RelayerConfig(privateKeysConfig.relayerConfigAlias)

	uniqueAddressesSet := make(map[string]struct{})

	r.GasOracleSenderPrivateKey, err = convertAndCheck(privateKeysConfig.GasOracleSenderPrivateKey, uniqueAddressesSet)
	if err != nil {
		return fmt.Errorf("error converting and checking gas oracle sender private key: %w", err)
	}

	r.CommitSenderPrivateKey, err = convertAndCheck(privateKeysConfig.CommitSenderPrivateKey, uniqueAddressesSet)
	if err != nil {
		return fmt.Errorf("error converting and checking commit sender private key: %w", err)
	}

	r.FinalizeSenderPrivateKey, err = convertAndCheck(privateKeysConfig.FinalizeSenderPrivateKey, uniqueAddressesSet)
	if err != nil {
		return fmt.Errorf("error converting and checking finalize sender private key: %w", err)
	}

	return nil
}

// MarshalJSON marshal RelayerConfig config, transfer private keys.
func (r *RelayerConfig) MarshalJSON() ([]byte, error) {
	privateKeysConfig := struct {
		relayerConfigAlias
		// The private key of the relayer
		GasOracleSenderPrivateKey string `json:"gas_oracle_sender_private_key"`
		CommitSenderPrivateKey    string `json:"commit_sender_private_key"`
		FinalizeSenderPrivateKey  string `json:"finalize_sender_private_key"`
	}{}

	privateKeysConfig.relayerConfigAlias = relayerConfigAlias(*r)
	privateKeysConfig.GasOracleSenderPrivateKey = common.Bytes2Hex(crypto.FromECDSA(r.GasOracleSenderPrivateKey))
	privateKeysConfig.CommitSenderPrivateKey = common.Bytes2Hex(crypto.FromECDSA(r.CommitSenderPrivateKey))
	privateKeysConfig.FinalizeSenderPrivateKey = common.Bytes2Hex(crypto.FromECDSA(r.FinalizeSenderPrivateKey))

	return json.Marshal(&privateKeysConfig)
}
