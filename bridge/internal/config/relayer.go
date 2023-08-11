package config

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"

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
	// The transaction type to use: LegacyTx, AccessListTx, DynamicFeeTx
	TxType string `json:"tx_type"`
	// The min balance set for check and set balance for sender's accounts.
	MinBalance *big.Int `json:"min_balance"`
	// The sender's pending count limit.
	PendingLimit int `json:"pending_limit"`
}

// RelayerConfig loads relayer configuration items.
// What we need to pay attention to is that
type RelayerConfig struct {
	// RollupContractAddress store the rollup contract address.
	RollupContractAddress common.Address `json:"rollup_contract_address,omitempty"`
	// MessengerContractAddress store the scroll messenger contract address.
	MessengerContractAddress common.Address `json:"messenger_contract_address"`
	// GasPriceOracleContractAddress store the scroll messenger contract address.
	GasPriceOracleContractAddress common.Address `json:"gas_price_oracle_contract_address"`
	// sender config
	SenderConfig *SenderConfig `json:"sender_config"`
	// gas oracle config
	GasOracleConfig *GasOracleConfig `json:"gas_oracle_config"`
	// The interval in which we send finalize batch transactions.
	FinalizeBatchIntervalSec uint64 `json:"finalize_batch_interval_sec"`
	// MessageRelayMinGasLimit to avoid OutOfGas error
	MessageRelayMinGasLimit uint64 `json:"message_relay_min_gas_limit,omitempty"`
	// The private key of the relayer
	MessageSenderPrivateKey   *ecdsa.PrivateKey `json:"-"`
	GasOracleSenderPrivateKey *ecdsa.PrivateKey `json:"-"`
	CommitSenderPrivateKey    *ecdsa.PrivateKey `json:"-"`
	FinalizeSenderPrivateKey  *ecdsa.PrivateKey `json:"-"`
}

// GasOracleConfig The config for updating gas price oracle.
type GasOracleConfig struct {
	// MinGasPrice store the minimum gas price to set.
	MinGasPrice uint64 `json:"min_gas_price"`
	// GasPriceDiff store the percentage of gas price difference.
	GasPriceDiff uint64 `json:"gas_price_diff"`
}

// relayerConfigAlias RelayerConfig alias name
type relayerConfigAlias RelayerConfig

// Helper function to convert and check private key uniqueness
func convertAndCheck(hexStr string, privKeys map[string]struct{}) (*ecdsa.PrivateKey, error) {
	// CommitSenderPrivateKey and FinalizeSenderPrivateKey are empty for l1 relayer.
	if hexStr == "" {
		return nil, nil
	}

	if _, exists := privKeys[hexStr]; exists {
		// DO NOT print private key here since it's confidential.
		return nil, fmt.Errorf("duplicate private key detected")
	}
	privKey, err := crypto.ToECDSA(common.FromHex(hexStr))
	if err != nil {
		return nil, fmt.Errorf("incorrect private_key format, err: %w", err)
	}
	privKeys[hexStr] = struct{}{}
	return privKey, nil
}

// UnmarshalJSON unmarshal relayer_config struct.
func (r *RelayerConfig) UnmarshalJSON(input []byte) error {
	var privateKeysConfig struct {
		relayerConfigAlias
		// The private key of the relayer
		MessageSenderPrivateKey   string `json:"message_sender_private_key"`
		GasOracleSenderPrivateKey string `json:"gas_oracle_sender_private_key"`
		CommitSenderPrivateKey    string `json:"commit_sender_private_key"`
		FinalizeSenderPrivateKey  string `json:"finalize_sender_private_key"`
	}
	if err := json.Unmarshal(input, &privateKeysConfig); err != nil {
		return err
	}

	*r = RelayerConfig(privateKeysConfig.relayerConfigAlias)

	existingPrivKeys := make(map[string]struct{})

	var err error
	r.MessageSenderPrivateKey, err = convertAndCheck(privateKeysConfig.MessageSenderPrivateKey, existingPrivKeys)
	if err != nil {
		return err
	}

	r.GasOracleSenderPrivateKey, err = convertAndCheck(privateKeysConfig.GasOracleSenderPrivateKey, existingPrivKeys)
	if err != nil {
		return err
	}

	r.CommitSenderPrivateKey, err = convertAndCheck(privateKeysConfig.CommitSenderPrivateKey, existingPrivKeys)
	if err != nil {
		return err
	}

	r.FinalizeSenderPrivateKey, err = convertAndCheck(privateKeysConfig.FinalizeSenderPrivateKey, existingPrivKeys)
	if err != nil {
		return err
	}

	return nil
}

// MarshalJSON marshal RelayerConfig config, transfer private keys.
func (r *RelayerConfig) MarshalJSON() ([]byte, error) {
	privateKeysConfig := struct {
		relayerConfigAlias
		// The private key of the relayer
		MessageSenderPrivateKey   string `json:"message_sender_private_key"`
		GasOracleSenderPrivateKey string `json:"gas_oracle_sender_private_key"`
		CommitSenderPrivateKey    string `json:"commit_sender_private_key"`
		FinalizeSenderPrivateKey  string `json:"finalize_sender_private_key"`
	}{}

	privateKeysConfig.relayerConfigAlias = relayerConfigAlias(*r)
	privateKeysConfig.MessageSenderPrivateKey = common.Bytes2Hex(crypto.FromECDSA(r.MessageSenderPrivateKey))
	privateKeysConfig.GasOracleSenderPrivateKey = common.Bytes2Hex(crypto.FromECDSA(r.GasOracleSenderPrivateKey))
	privateKeysConfig.CommitSenderPrivateKey = common.Bytes2Hex(crypto.FromECDSA(r.CommitSenderPrivateKey))
	privateKeysConfig.FinalizeSenderPrivateKey = common.Bytes2Hex(crypto.FromECDSA(r.FinalizeSenderPrivateKey))

	return json.Marshal(&privateKeysConfig)
}
