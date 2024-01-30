package types

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// TransactionData represents custom transaction structure for forward compatibility.
type TransactionData struct {
	Type     uint8           `json:"type"`
	Nonce    uint64          `json:"nonce"`
	Gas      uint64          `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	To       *common.Address `json:"to"`
	Value    *hexutil.Big    `json:"value"`
	Data     string          `json:"data"`
	V        *hexutil.Big    `json:"v"`
	R        *hexutil.Big    `json:"r"`
	S        *hexutil.Big    `json:"s"`
}

// DecodeTransactions converts hex-encoded RLP transactions to types.Transaction slice.
func DecodeTransactions(encodedTx string) ([]*types.Transaction, error) {
	var transactions []*types.Transaction
	// Try decoding as RLP-encoded transactions first
	bytes, err := base64.StdEncoding.DecodeString(encodedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 string, err: %w", err)
	}
	rlpErr := rlp.DecodeBytes(bytes, &transactions)
	if rlpErr == nil {
		return transactions, nil
	}

	// RLP decoding failed, then attempt to unmarshal into TransactionData
	var txData []*TransactionData
	if jsonErr := json.Unmarshal([]byte(encodedTx), &txData); jsonErr != nil {
		return nil, fmt.Errorf("fatal: both RLP decode and JSON decode are failed, rlpErr: %w, jsonErr: %w", rlpErr, jsonErr)
	}

	transactions, err = convertTxDataToTxs(txData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert transaction data to transactions, err: %w", err)
	}

	if len(transactions) != len(txData) {
		return nil, fmt.Errorf("validation failed: the number of decoded transactions (%d) does not match the number of transaction data entries (%d)", len(transactions), len(txData))
	}

	return transactions, nil
}

// convertTxDataToTxs converts a slice of TransactionData to a slice of *types.Transaction.
func convertTxDataToTxs(txData []*TransactionData) ([]*types.Transaction, error) {
	var txs []*types.Transaction

	for _, oldTx := range txData {
		data, err := hexutil.Decode(oldTx.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction data field, error: %w", err)
		}

		// The l2geth only supports legacy and l1 message transactions before enabling EIP-1559.
		switch oldTx.Type {
		case types.LegacyTxType:
			newTx := types.NewTx(&types.LegacyTx{
				Nonce:    oldTx.Nonce,
				To:       oldTx.To,
				Value:    oldTx.Value.ToInt(),
				Gas:      oldTx.Gas,
				GasPrice: oldTx.GasPrice.ToInt(),
				Data:     data,
				V:        oldTx.V.ToInt(),
				R:        oldTx.R.ToInt(),
				S:        oldTx.S.ToInt(),
			})
			txs = append(txs, newTx)

		case types.L1MessageTxType:
			newTx := types.NewTx(&types.L1MessageTx{
				To:         oldTx.To,
				Value:      oldTx.Value.ToInt(),
				Gas:        oldTx.Gas,
				Data:       data,
				QueueIndex: oldTx.Nonce,
			})
			txs = append(txs, newTx)

		default:
			return nil, fmt.Errorf("unexpected tx type: %v", oldTx.Type)
		}
	}

	return txs, nil
}
