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

// Encode encodes the WrappedBlock into RollupV2 BlockContext Encoding.
func (w *WrappedBlock) Encode() ([]byte, error) {
	bytes := make([]byte, 60)

	if !w.Header.Number.IsUint64() {
		return nil, errors.New("block number is not uint64")
	}

	if len(w.Transactions) > math.MaxUint16 {
		return nil, errors.New("number of transactions exceeds max uint16")
	}

	binary.BigEndian.PutUint64(bytes[0:], w.Header.Number.Uint64())

	binary.BigEndian.PutUint64(bytes[8:], w.Header.Time)

	// TODO: Currently, baseFee is 0

	binary.BigEndian.PutUint64(bytes[48:], w.Header.GasLimit)

	binary.BigEndian.PutUint16(bytes[56:], uint16(len(w.Transactions)))

	// TODO: set numL1Messages properly

	return bytes, nil
}
