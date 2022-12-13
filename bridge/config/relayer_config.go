package config

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"

	apollo_config "scroll-tech/common/apollo"
)

// SenderConfig The config for transaction sender
type SenderConfig struct {
	// The sender type
	SenderSide `json:"sender_side"`
	// The RPC endpoint of the ethereum or scroll public node.
	Endpoint string `json:"endpoint"`
	// The transaction type to use: LegacyTx, AccessListTx, DynamicFeeTx
	TxType string `json:"tx_type"`
	// Skip confirmation (for unit test)
	SkipConfirmation bool
}

// GetCheckPendingTime : get time to trigger check pending txs in sender.
func (s *SenderConfig) GetCheckPendingTime() uint64 {
	if s.SenderSide == L1Sender {
		return uint64(apollo_config.AgolloClient.GetIntValue("l1SenderCheckPendingTime", 3))
	}
	return uint64(apollo_config.AgolloClient.GetIntValue("l2SenderCheckPendingTime", 10))
}

// GetConfirmations : get the gap number between a block be confirmed and the latest block.
func (s *SenderConfig) GetConfirmations() uint64 {
	if s.SkipConfirmation {
		return 0
	} else if s.SenderSide == L1Sender {
		return uint64(apollo_config.AgolloClient.GetIntValue("l2Confirmations", 1))
	}
	return uint64(apollo_config.AgolloClient.GetIntValue("l1Confirmations", 6))
}

// GetEscalateBlocks : get the number of blocks to wait to escalate increase gas price of the transaction.
func (s *SenderConfig) GetEscalateBlocks() uint64 {
	return uint64(apollo_config.AgolloClient.GetIntValue("escalateBlocks", 100))
}

// GetMinBalance : get the min balance set for check and set balance for sender's accounts.
func (s *SenderConfig) GetMinBalance() *big.Int {
	minBalanceStr := apollo_config.AgolloClient.GetStringValue("minBalance", "100000000000000000000")
	minBalance, ok := new(big.Int).SetString(minBalanceStr, 10)
	if ok {
		return minBalance
	}
	minBalance.SetString("100000000000000000000", 10)
	return minBalance
}

// GetEscalateMultipleNum : get the numerator of gas price escalate multiple.
func (s *SenderConfig) GetEscalateMultipleNum() uint64 {
	return uint64(apollo_config.AgolloClient.GetIntValue("escalateMultipleNum", 11))
}

// GetEscalateMultipleDen : get the denominator of gas price escalate multiple.
func (s *SenderConfig) GetEscalateMultipleDen() uint64 {
	return uint64(apollo_config.AgolloClient.GetIntValue("escalateMultipleDen", 10))
}

// GetMaxGasPrice : get the maximum gas price can be used to send transaction.
func (s *SenderConfig) GetMaxGasPrice() uint64 {
	maxGasPriceStr := apollo_config.AgolloClient.GetStringValue("maxGasPrice", "10000000000")
	maxGasPrice, err := strconv.ParseInt(maxGasPriceStr, 10, 64)
	if err != nil {
		return 10000000000
	}
	return uint64(maxGasPrice)
}

// SenderSide sender type (L1Sender, L2Sender)
type SenderSide int

const (
	_ SenderSide = iota
	// L1Sender : sender of l1 relayer
	L1Sender
	// L2Sender : sender of l2 relayer
	L2Sender
)

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

// relayerConfigAlias RelayerConfig alias name
type relayerConfigAlias RelayerConfig

// UnmarshalJSON unmarshal relayer_config struct.
func (r *RelayerConfig) UnmarshalJSON(input []byte) error {
	var jsonConfig struct {
		relayerConfigAlias
		// The private key of the relayer
		MessageSenderPrivateKeys []string `json:"message_sender_private_keys"`
		RollupSenderPrivateKeys  []string `json:"rollup_sender_private_keys,omitempty"`
	}
	if err := json.Unmarshal(input, &jsonConfig); err != nil {
		return err
	}

	// Get messenger private key list.
	*r = RelayerConfig(jsonConfig.relayerConfigAlias)
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

// MarshalJSON marshal RelayerConfig config, transfer private keys.
func (r *RelayerConfig) MarshalJSON() ([]byte, error) {
	jsonConfig := struct {
		relayerConfigAlias
		// The private key of the relayer
		MessageSenderPrivateKeys []string `json:"message_sender_private_keys"`
		RollupSenderPrivateKeys  []string `json:"rollup_sender_private_keys,omitempty"`
	}{relayerConfigAlias(*r), nil, nil}

	// Transfer message sender private keys to hex type.
	for _, priv := range r.MessageSenderPrivateKeys {
		jsonConfig.MessageSenderPrivateKeys = append(jsonConfig.MessageSenderPrivateKeys, common.Bytes2Hex(crypto.FromECDSA(priv)))
	}

	// Transfer rollup sender private keys to hex type.
	for _, priv := range r.RollupSenderPrivateKeys {
		jsonConfig.RollupSenderPrivateKeys = append(jsonConfig.RollupSenderPrivateKeys, common.Bytes2Hex(crypto.FromECDSA(priv)))
	}

	return json.Marshal(&jsonConfig)
}
