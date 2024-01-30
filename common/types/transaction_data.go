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

// TransactionData defines a structure compatible with legacy plaintext JSON transaction data.
// This is used for backward compatibility.
type TransactionData struct {
	Type     uint8           `json:"type"`
	Nonce    uint64          `json:"nonce"`
	Gas      uint64          `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Value    *hexutil.Big    `json:"value"`
	Data     string          `json:"data"`
	V        *hexutil.Big    `json:"v"`
	R        *hexutil.Big    `json:"r"`
	S        *hexutil.Big    `json:"s"`
}

// DecodeTransactions decodes hex strings into a slice of types.Transaction.
func DecodeTransactions(encodedTx string) ([]*types.Transaction, error) {
	var transactions []*types.Transaction
	// Decode the base64 string to bytes as RLP-encoded transactions
	bytes, err := base64.StdEncoding.DecodeString(encodedTx)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}
	rlpErr := rlp.DecodeBytes(bytes, &transactions)
	if rlpErr == nil {
		return transactions, nil
	}

	// If RLP decoding fails, try to unmarshal into TransactionData
	var txData []*TransactionData
	if jsonErr := json.Unmarshal([]byte(encodedTx), &txData); jsonErr != nil {
		return nil, fmt.Errorf("RLP and JSON decode failure: rlpErr: %w, jsonErr: %w", rlpErr, jsonErr)
	}

	// Convert TransactionData to types.Transaction
	transactions, err = convertTxDataToTxs(txData)
	if err != nil {
		return nil, fmt.Errorf("conversion of TransactionData failed: %w", err)
	}

	// Validate the number of decoded transactions
	if len(transactions) != len(txData) {
		return nil, fmt.Errorf("decoded transaction count mismatch: got %d, want %d", len(transactions), len(txData))
	}

	return transactions, nil
}

// convertTxDataToTxs transforms []*TransactionData into []*types.Transaction.
func convertTxDataToTxs(txData []*TransactionData) ([]*types.Transaction, error) {
	var txs []*types.Transaction

	for _, oldTx := range txData {
		data, err := hexutil.Decode(oldTx.Data)
		if err != nil {
			return nil, fmt.Errorf("hex decode of 'data' field failed: %w", err)
		}

		// Handle specific transaction types, considering EIP-1559 is not in use.
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
				Sender:     oldTx.From,
			})
			txs = append(txs, newTx)

		default:
			return nil, fmt.Errorf("unsupported tx type: %v", oldTx.Type)
		}
	}

	return txs, nil
}
