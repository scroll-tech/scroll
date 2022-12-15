package config

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
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
		return GetL2CheckPendingTime()
	}
	return GetL1CheckPendingTime()
}

// GetConfirmations : get the gap number between a block be confirmed and the latest block.
func (s *SenderConfig) GetConfirmations() uint64 {
	if s.SkipConfirmation {
		// for unit test: skip confirmation.
		return 0
	} else if s.SenderSide == L1Sender {
		return GetL2Confirmations()
	}
	return GetL1Confirmations()
}

// SenderSide sender type (L1Sender, L2Sender).
type SenderSide int

const (
	_ SenderSide = iota
	// L1Sender : sender of l1 relayer.
	L1Sender
	// L2Sender : sender of l2 relayer.
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
