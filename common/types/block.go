package types

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

// CalldataNonZeroByteGas is the gas consumption per non zero byte in calldata.
const CalldataNonZeroByteGas = 16

// GetKeccak256Gas calculates keccak256 hash gas.
func GetKeccak256Gas(size uint64) uint64 {
	return 30 + 6*((size+31)/32) // 30 + 6 * ceil(size / 32)
}

// WrappedBlock contains the block's Header, Transactions and WithdrawTrieRoot hash.
type WrappedBlock struct {
	Header *types.Header `json:"header"`
	// Transactions is only used for recover types.Transactions, the from of types.TransactionData field is missing.
	Transactions         []*types.TransactionData `json:"transactions"`
	WithdrawRoot         common.Hash              `json:"withdraw_trie_root,omitempty"`
	RowConsumption       *types.RowConsumption    `json:"row_consumption"`
	txPayloadLengthCache map[string]uint64
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

	// note: numL1Messages includes skipped messages
	numL1Messages := w.NumL1Messages(totalL1MessagePoppedBefore)
	if numL1Messages > math.MaxUint16 {
		return nil, errors.New("number of L1 messages exceeds max uint16")
	}

	// note: numTransactions includes skipped messages
	numL2Transactions := w.L2TxsNum()
	numTransactions := numL1Messages + numL2Transactions
	if numTransactions > math.MaxUint16 {
		return nil, errors.New("number of transactions exceeds max uint16")
	}

	binary.BigEndian.PutUint64(bytes[0:], w.Header.Number.Uint64())
	binary.BigEndian.PutUint64(bytes[8:], w.Header.Time)
	// TODO: [16:47] Currently, baseFee is 0, because we disable EIP-1559.
	binary.BigEndian.PutUint64(bytes[48:], w.Header.GasLimit)
	binary.BigEndian.PutUint16(bytes[56:], uint16(numTransactions))
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
		size += 64 // 60 bytes BlockContext + 4 bytes payload length
		size += w.getTxPayloadLength(txData)
	}
	return size
}

// EstimateL1CommitGas calculates the total L1 commit gas for this block approximately.
func (w *WrappedBlock) EstimateL1CommitGas() uint64 {
	var total uint64
	var numL1Messages uint64
	for _, txData := range w.Transactions {
		if txData.Type == types.L1MessageTxType {
			numL1Messages++
			continue
		}

		txPayloadLength := w.getTxPayloadLength(txData)
		total += CalldataNonZeroByteGas * txPayloadLength // an over-estimate: treat each byte as non-zero
		total += CalldataNonZeroByteGas * 64              // 60 bytes BlockContext + 4 bytes payload length
		total += GetKeccak256Gas(txPayloadLength)         // l2 tx hash
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

func (w *WrappedBlock) getTxPayloadLength(txData *types.TransactionData) uint64 {
	if w.txPayloadLengthCache == nil {
		w.txPayloadLengthCache = make(map[string]uint64)
	}

	if length, exists := w.txPayloadLengthCache[txData.TxHash]; exists {
		return length
	}

	rlpTxData, err := convertTxDataToRLPEncoding(txData)
	if err != nil {
		log.Crit("convertTxDataToRLPEncoding failed, which should not happen", "hash", txData.TxHash, "err", err)
		return 0
	}
	txPayloadLength := uint64(len(rlpTxData))
	w.txPayloadLengthCache[txData.TxHash] = txPayloadLength
	return txPayloadLength
}

func convertTxDataToRLPEncoding(txData *types.TransactionData) ([]byte, error) {
	data, err := hexutil.Decode(txData.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode txData.Data: %s, err: %w", txData.Data, err)
	}

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

	rlpTxData, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal binary of the tx: %+v, err: %w", tx, err)
	}

	return rlpTxData, nil
}
