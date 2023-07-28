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
	// The interval (in seconds) to check balance and top up sender's accounts
	CheckBalanceTime uint64 `json:"check_balance_time"`
	// The sender's pending count limit.
	PendingLimit int `json:"pending_limit,omitempty"`
}

// RelayerConfig loads relayer configuration items.
// What we need to pay attention to is that
// `MessageSenderPrivateKeys` and `RollupSenderPrivateKeys` cannot have common private keys.
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
	MessageSenderPrivateKeys   []*ecdsa.PrivateKey `json:"-"`
	GasOracleSenderPrivateKeys []*ecdsa.PrivateKey `json:"-"`
	RollupSenderPrivateKeys    []*ecdsa.PrivateKey `json:"-"`
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

// UnmarshalJSON unmarshal relayer_config struct.
func (r *RelayerConfig) UnmarshalJSON(input []byte) error {
	var jsonConfig struct {
		relayerConfigAlias
		// The private key of the relayer
		MessageSenderPrivateKeys   []string `json:"message_sender_private_keys"`
		GasOracleSenderPrivateKeys []string `json:"gas_oracle_sender_private_keys"`
		RollupSenderPrivateKeys    []string `json:"rollup_sender_private_keys,omitempty"`
	}
	if err := json.Unmarshal(input, &jsonConfig); err != nil {
		return err
	}

	*r = RelayerConfig(jsonConfig.relayerConfigAlias)

	// Get messenger private key list.
	for _, privStr := range jsonConfig.MessageSenderPrivateKeys {
		priv, err := crypto.ToECDSA(common.FromHex(privStr))
		if err != nil {
			return fmt.Errorf("incorrect private_key_list format, err: %v", err)
		}
		r.MessageSenderPrivateKeys = append(r.MessageSenderPrivateKeys, priv)
	}

	// Get gas oracle private key list.
	for _, privStr := range jsonConfig.GasOracleSenderPrivateKeys {
		priv, err := crypto.ToECDSA(common.FromHex(privStr))
		if err != nil {
			return fmt.Errorf("incorrect private_key_list format, err: %v", err)
		}
		r.GasOracleSenderPrivateKeys = append(r.GasOracleSenderPrivateKeys, priv)
	}

	// Get rollup private key
	for _, privStr := range jsonConfig.RollupSenderPrivateKeys {
		priv, err := crypto.ToECDSA(common.FromHex(privStr))
		if err != nil {
			return fmt.Errorf("incorrect prover_private_key format, err: %v", err)
		}
		r.RollupSenderPrivateKeys = append(r.RollupSenderPrivateKeys, priv)
	}

	return nil
}

// MarshalJSON marshal RelayerConfig config, transfer private keys.
func (r *RelayerConfig) MarshalJSON() ([]byte, error) {
	jsonConfig := struct {
		relayerConfigAlias
		// The private key of the relayer
		MessageSenderPrivateKeys   []string `json:"message_sender_private_keys"`
		GasOracleSenderPrivateKeys []string `json:"gas_oracle_sender_private_keys,omitempty"`
		RollupSenderPrivateKeys    []string `json:"rollup_sender_private_keys,omitempty"`
	}{relayerConfigAlias(*r), nil, nil, nil}

	// Transfer message sender private keys to hex type.
	for _, priv := range r.MessageSenderPrivateKeys {
		jsonConfig.MessageSenderPrivateKeys = append(jsonConfig.MessageSenderPrivateKeys, common.Bytes2Hex(crypto.FromECDSA(priv)))
	}

	// Transfer rollup sender private keys to hex type.
	for _, priv := range r.GasOracleSenderPrivateKeys {
		jsonConfig.GasOracleSenderPrivateKeys = append(jsonConfig.GasOracleSenderPrivateKeys, common.Bytes2Hex(crypto.FromECDSA(priv)))
	}

	// Transfer rollup sender private keys to hex type.
	for _, priv := range r.RollupSenderPrivateKeys {
		jsonConfig.RollupSenderPrivateKeys = append(jsonConfig.RollupSenderPrivateKeys, common.Bytes2Hex(crypto.FromECDSA(priv)))
	}

	return json.Marshal(&jsonConfig)
}
