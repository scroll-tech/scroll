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

// GetKeccak256Gas calculates the gas cost for computing the keccak256 hash of a given size.
func GetKeccak256Gas(size uint64) uint64 {
	return GetMemoryExpansionCost(size) + 30 + 6*((size+31)/32)
}

// GetMemoryExpansionCost calculates the cost of memory expansion for a given memoryByteSize.
func GetMemoryExpansionCost(memoryByteSize uint64) uint64 {
	memorySizeWord := (memoryByteSize + 31) / 32
	memoryCost := (memorySizeWord*memorySizeWord)/512 + (3 * memorySizeWord)
	return memoryCost
}

// WrappedBlock contains the block's Header, Transactions, WithdrawTrieRoot hash and LastAppliedL1Block.
type WrappedBlock struct {
	Header *types.Header `json:"header"`
	// Transactions is only used for recover types.Transactions, the from of types.TransactionData field is missing.
	Transactions         []*types.TransactionData `json:"transactions"`
	WithdrawRoot         common.Hash              `json:"withdraw_trie_root,omitempty"`
	RowConsumption       *types.RowConsumption    `json:"row_consumption"`
	LastAppliedL1Block   uint64                   `json:"last_applied_l1_block"`
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

// NumL2Transactions returns the number of L2 transactions in this block.
func (w *WrappedBlock) NumL2Transactions() uint64 {
	var count uint64
	for _, txData := range w.Transactions {
		if txData.Type != types.L1MessageTxType {
			count++
		}
	}
	return count
}

// Encode encodes the WrappedBlock into RollupV2 BlockContext Encoding.
func (w *WrappedBlock) Encode(totalL1MessagePoppedBefore uint64) ([]byte, error) {
	bytes := make([]byte, 68)

	if !w.Header.Number.IsUint64() {
		return nil, errors.New("block number is not uint64")
	}

	// note: numL1Messages includes skipped messages
	numL1Messages := w.NumL1Messages(totalL1MessagePoppedBefore)
	if numL1Messages > math.MaxUint16 {
		return nil, errors.New("number of L1 messages exceeds max uint16")
	}

	// note: numTransactions includes skipped messages
	numL2Transactions := w.NumL2Transactions()
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
	binary.BigEndian.PutUint64(bytes[60:], w.LastAppliedL1Block)

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
		size += 4 // 4 bytes payload length
		size += w.getTxPayloadLength(txData)
	}
	size += 68 //  68 bytes BlockContext
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
		total += CalldataNonZeroByteGas * 4               // 4 bytes payload length
		total += GetKeccak256Gas(txPayloadLength)         // l2 tx hash
	}

	// 68 bytes BlockContext calldata
	total += CalldataNonZeroByteGas * 68

	// sload
	total += 2100 * numL1Messages // numL1Messages times cold sload in L1MessageQueue

	// staticcall
	total += 100 * numL1Messages // numL1Messages times call to L1MessageQueue
	total += 100 * numL1Messages // numL1Messages times warm address access to L1MessageQueue

	total += GetMemoryExpansionCost(36) * numL1Messages // staticcall to proxy
	total += 100 * numL1Messages                        // read admin in proxy
	total += 100 * numL1Messages                        // read impl in proxy
	total += 100 * numL1Messages                        // access impl
	total += GetMemoryExpansionCost(36) * numL1Messages // delegatecall to impl

	return total
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
