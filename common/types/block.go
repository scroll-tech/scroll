package types

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// WrappedBlock contains the block's Header, Transactions and WithdrawTrieRoot hash.
type WrappedBlock struct {
	Header *types.Header `json:"header"`
	// Transactions is only used for recover types.Transactions, the from of types.TransactionData field is missing.
	Transactions []*types.TransactionData `json:"transactions"`
	WithdrawRoot common.Hash              `json:"withdraw_trie_root,omitempty"`
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

// EstimateL1CommitGas calculates the total L1 commit gas for this block approximately.
func (w *WrappedBlock) EstimateL1CommitGas() uint64 {
	getKeccakGas := func(size uint64) uint64 {
		return 30 + 6*((size+31)/32) // 30 + 6 * ceil(size / 32)
	}

	var total uint64
	var numL1Messages uint64
	for _, txData := range w.Transactions {
		if txData.Type == types.L1MessageTxType {
			numL1Messages++
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
		txPayloadLength := uint64(len(rlpTxData))
		total += 16 * uint64(txPayloadLength)  // an over-estimate: treat each byte as non-zero
		total += 16 * 4                        // size of a uint32 field
		total += getKeccakGas(txPayloadLength) // l2 tx hash
	}

	// sload
	total += 2100 * numL1Messages // numL1Messages times cold sload in L1MessageQueue

	// staticcall
	total += 100 * numL1Messages // numL1Messages times call to L1MessageQueue
	total += 100 * numL1Messages // numL1Messages times warm address access to L1MessageQueue

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
