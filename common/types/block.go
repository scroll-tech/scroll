package types

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// WrappedBlock contains the block's Header, Transactions and WithdrawTrieRoot hash.
type WrappedBlock struct {
	Header *types.Header `json:"header"`
	// Transactions is only used for recover types.Transactions, the from of types.TransactionData field is missing.
	Transactions     []*types.TransactionData `json:"transactions"`
	WithdrawTrieRoot common.Hash              `json:"withdraw_trie_root,omitempty"`
}

func(w *WrappedBlock) Encode() ([]byte, error) {
	bytes := make([]byte, 0)

	if !w.Header.Number.IsUint64() {
		return nil, errors.New("block number is not uint64")
	}

	println("bytes: ", bytes)

	numberBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(numberBytes, w.Header.Number.Uint64())
	bytes = append(bytes, numberBytes...)

	println("bytes: ", bytes)

	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, w.Header.Time)
	bytes = append(bytes, timeBytes...)

	println("bytes: ", bytes)

	bytes = append(bytes, make([]byte, 32)...) // Currently, baseFee is 0

	println("bytes: ", bytes)

	gasLimitBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(gasLimitBytes, w.Header.GasLimit)
	bytes = append(bytes, gasLimitBytes...)

	println("bytes: ", bytes)

	if len(w.Transactions) > math.MaxUint16 {
		return nil, errors.New("number of transactions exceeds max uint16")
	}

	numTransactionsBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(numTransactionsBytes, uint16(len(w.Transactions)))
	bytes = append(bytes, numTransactionsBytes...)

	println("bytes: ", bytes)

	bytes = append(bytes, 0,0) // Currently, numL1Messages is 0

	println("bytes: ", bytes)
	
	return bytes, nil
}