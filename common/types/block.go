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

// NumL1Messages returns the number of L1 messages in this block.
// This number is the sum of included and skipped L1 messages.
func (w *WrappedBlock) NumL1Messages(totalL1MessagePoppedBefore uint64) uint64 {
	var lastQueueIndex *uint64
	for _, txData := range w.Transactions {
		if txData.Type == types.L1MessageTxType {
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

// EstimateL1CommitCalldataSize calculates the calldata size in l1 commit approximately.
// TODO: The calculation could be more accurate by using 58 + len(l2TxDataBytes) (see Chunk).
// This needs to be adjusted in the future.
func (w *WrappedBlock) EstimateL1CommitCalldataSize() uint64 {
	var size uint64
	for _, txData := range w.Transactions {
		if txData.Type == types.L1MessageTxType {
			continue
		}
		size += uint64(len(txData.Data))
	}
	return size
}

// EstimateL1CommitGas calculates the calldata gas in l1 commit approximately.
func (w *WrappedBlock) EstimateL1CommitGas() uint64 {
	var total uint64
	var numL1Messages uint64
	for _, txData := range w.Transactions {
		if txData.Type == types.L1MessageTxType {
			numL1Messages++
			continue
		}
	}

	// sload
	total += numL1Messages * 2100 // numL1Messages times cold sload in L1MessageQueue

	// staticcall
	total += numL1Messages * 100 // numL1Messages times call to L1MessageQueue
	total += numL1Messages * 100 // numL1Messages times warm address access to L1MessageQueue

	total += w.EstimateL1CommitCalldataSize() * 16 // approximate calldata gas cost

	return total
}

// L2TxsNum calculates the number of l2 txs.
func (w *WrappedBlock) L2TxsNum() uint64 {
	var count uint64
	for _, txData := range w.Transactions {
		if txData.Type != types.L1MessageTxType {
			count++
		}
	}
	return count
}
