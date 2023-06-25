package types

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// L1MessageTxType represents l2geth's l1 message tx type.
// TODO: replace this with geth version after new version is released.
const L1MessageTxType = 0x7E

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
		if txData.Type == L1MessageTxType {
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

// ApproximateL1CommitCalldataSize calculates the calldata size in l1 commit approximately.
// TODO: The calculation could be more accurate by using 58 + len(l2TxDataBytes) (see Chunk).
// This needs to be adjusted in the future.
func (w *WrappedBlock) ApproximateL1CommitCalldataSize() uint64 {
	var size uint64
	for _, tx := range w.Transactions {
		size += uint64(len(tx.Data))
	}
	return size
}

const nonZeroByteGas uint64 = 16
const zeroByteGas uint64 = 4

// ApproximateL1CommitGas calculates the calldata gas in l1 commit approximately.
// TODO: This will need to be adjusted.
// The part added here is only the calldata cost,
// but we have execution cost for verifying blocks / chunks / batches and storing the batch hash.
func (w *WrappedBlock) ApproximateL1CommitGas() uint64 {
	var total uint64
	for _, txData := range w.Transactions {
		if txData.Type == L1MessageTxType {
			continue
		}
		data, _ := hexutil.Decode(txData.Data)
		tx := types.NewTx(&types.LegacyTx{
			Nonce:    txData.Nonce,
			To:       txData.To,
			Value:    txData.Value.ToInt(),
			Gas:      txData.Gas,
			GasPrice: txData.GasPrice.ToInt(),
			Data:     data,
			V:        txData.V.ToInt(),
			R:        txData.R.ToInt(),
			S:        txData.S.ToInt(),
		})
		rlpTxData, _ := tx.MarshalBinary()

		for _, b := range rlpTxData {
			if b == 0 {
				total += zeroByteGas
			} else {
				total += nonZeroByteGas
			}
		}

		var txLen [4]byte
		binary.BigEndian.PutUint32(txLen[:], uint32(len(rlpTxData)))

		for _, b := range txLen {
			if b == 0 {
				total += zeroByteGas
			} else {
				total += nonZeroByteGas
			}
		}
	}
	return total
}

// GetL2TxsNum calculates the number of l2 txs.
func (w *WrappedBlock) GetL2TxsNum() uint64 {
	var count uint64
	for _, txData := range w.Transactions {
		if txData.Type != L1MessageTxType {
			count++
		}
	}
	return count
}
