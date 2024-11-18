package config

import (
	"github.com/scroll-tech/go-ethereum/common"
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
	// The maximum number of pending blob-carrying transactions
	MaxPendingBlobTxs int64 `json:"max_pending_blob_txs"`
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

	// Configs of transaction signers (GasOracle, Commit, Finalize)
	GasOracleSenderSignerConfig *SignerConfig `json:"gas_oracle_sender_signer_config"`
	CommitSenderSignerConfig    *SignerConfig `json:"commit_sender_signer_config"`
	FinalizeSenderSignerConfig  *SignerConfig `json:"finalize_sender_signer_config"`

	// Indicates if bypass features specific to testing environments are enabled.
	EnableTestEnvBypassFeatures bool `json:"enable_test_env_bypass_features"`
	// The timeout in seconds for finalizing a batch without proof, only used when EnableTestEnvBypassFeatures is true.
	FinalizeBatchWithoutProofTimeoutSec uint64 `json:"finalize_batch_without_proof_timeout_sec"`
	// The timeout in seconds for finalizing a bundle without proof, only used when EnableTestEnvBypassFeatures is true.
	FinalizeBundleWithoutProofTimeoutSec uint64 `json:"finalize_bundle_without_proof_timeout_sec"`
}

// AlternativeGasTokenConfig The configuration for handling token exchange rates when updating the gas price oracle.
type AlternativeGasTokenConfig struct {
	Enabled           bool    `json:"enabled"`
	Mode              string  `json:"mode"`
	FixedExchangeRate float64 `json:"fixed_exchange_rate"` // fixed exchange rate of L2 gas token / L1 gas token
	TokenSymbolPair   string  `json:"token_symbol_pair"`   // The pair should be L2 gas token symbol + L1 gas token symbol
}

// GasOracleConfig The config for updating gas price oracle.
type GasOracleConfig struct {
	// MinGasPrice store the minimum gas price to set.
	MinGasPrice uint64 `json:"min_gas_price"`
	// GasPriceDiff is the minimum percentage of gas price difference to update gas oracle.
	GasPriceDiff uint64 `json:"gas_price_diff"`
	// AlternativeGasTokenConfig The configuration for handling token exchange rates when updating the gas price oracle.
	AlternativeGasTokenConfig *AlternativeGasTokenConfig `json:"alternative_gas_token_config"`

	// CheckCommittedBatchesWindowMinutes the time frame to check if we committed batches to decide to update gas oracle or not in minutes
	CheckCommittedBatchesWindowMinutes int    `json:"check_committed_batches_window_minutes"`
	L1BaseFeeDefault                   uint64 `json:"l1_base_fee_default"`
	L1BlobBaseFeeDefault               uint64 `json:"l1_blob_base_fee_default"`

	// L1BlobBaseFeeThreshold the threshold of L1 blob base fee to enter the default gas price mode
	L1BlobBaseFeeThreshold uint64 `json:"l1_blob_base_fee_threshold"`
}

// SignerConfig - config of signer, contains type and config corresponding to type
type SignerConfig struct {
	SignerType             string                  `json:"signer_type"` // type of signer can be PrivateKey or RemoteSigner
	PrivateKeySignerConfig *PrivateKeySignerConfig `json:"private_key_signer_config"`
	RemoteSignerConfig     *RemoteSignerConfig     `json:"remote_signer_config"`
}

// PrivateKeySignerConfig - config of private signer, contains private key
type PrivateKeySignerConfig struct {
	PrivateKey string `json:"private_key"` // private key of signer in case of PrivateKey signerType
}

// RemoteSignerConfig - config of private signer, contains address and remote URL
type RemoteSignerConfig struct {
	RemoteSignerUrl string `json:"remote_signer_url"` // remote signer url (web3signer) in case of RemoteSigner signerType
	SignerAddress   string `json:"signer_address"`    // address of signer
}
