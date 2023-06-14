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

func (w *WrappedBlock) NumL1Messages(totalL1MessagePoppedBefore uint64) uint64 {
	var lastQueueIndex *uint64
	for _, txData := range w.Transactions {
		if txData.Type == 0x7E {
			lastQueueIndex = &txData.Nonce
		}
	}
	if lastQueueIndex == nil {
		return 0
	}
	// note: last queue index included before this block is totalL1MessagePoppedBefore - 1
	// TODO: cache results
	return *lastQueueIndex - totalL1MessagePoppedBefore + 1
}

// Encode encodes the WrappedBlock into RollupV2 BlockContext Encoding.
func (w *WrappedBlock) Encode(totalL1MessagePoppedBefore uint64) ([]byte, error) {
	bytes := make([]byte, 60)

	if !w.Header.Number.IsUint64() {
		return nil, errors.New("block number is not uint64")
	}
	if len(w.Transactions) > math.MaxUint16 {
		return nil, errors.New("number of transactions exceeds max uint16")
	}

	numL1Messages := w.NumL1Messages(totalL1MessagePoppedBefore)
	if numL1Messages > math.MaxUint16 {
		return nil, errors.New("number of L1 messages exceeds max uint16")
	}

	binary.BigEndian.PutUint64(bytes[0:], w.Header.Number.Uint64())
	binary.BigEndian.PutUint64(bytes[8:], w.Header.Time)
	// TODO: [16:47] Currently, baseFee is 0, because we disable EIP-1559.
	binary.BigEndian.PutUint64(bytes[48:], w.Header.GasLimit)
	binary.BigEndian.PutUint16(bytes[56:], uint16(len(w.Transactions)))
	binary.BigEndian.PutUint16(bytes[58:], uint16(numL1Messages))

	return bytes, nil
}
