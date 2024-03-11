package encoding

import (
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

// Block represents an L2 block.
type Block struct {
	Header         *types.Header
	Transactions   []*types.TransactionData
	WithdrawRoot   common.Hash           `json:"withdraw_trie_root,omitempty"`
	RowConsumption *types.RowConsumption `json:"row_consumption,omitempty"`
}

// Chunk represents a group of blocks.
type Chunk struct {
	Blocks []*Block `json:"blocks"`
}

// Batch represents a batch of chunks.
type Batch struct {
	Index                      uint64
	TotalL1MessagePoppedBefore uint64
	ParentBatchHash            common.Hash
	Chunks                     []*Chunk

	// Only used in updating db info.
	StartChunkIndex uint64
	EndChunkIndex   uint64
	StartChunkHash  common.Hash
	EndChunkHash    common.Hash
}

// NumL1Messages returns the number of L1 messages in this block.
// This number is the sum of included and skipped L1 messages.
func (b *Block) NumL1Messages(totalL1MessagePoppedBefore uint64) uint64 {
	var lastQueueIndex *uint64
	for _, txData := range b.Transactions {
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
func (b *Block) NumL2Transactions() uint64 {
	var count uint64
	for _, txData := range b.Transactions {
		if txData.Type != types.L1MessageTxType {
			count++
		}
	}
	return count
}

// NumL1Messages returns the number of L1 messages in this chunk.
// This number is the sum of included and skipped L1 messages.
func (c *Chunk) NumL1Messages(totalL1MessagePoppedBefore uint64) uint64 {
	var numL1Messages uint64
	for _, block := range c.Blocks {
		numL1MessagesInBlock := block.NumL1Messages(totalL1MessagePoppedBefore)
		numL1Messages += numL1MessagesInBlock
		totalL1MessagePoppedBefore += numL1MessagesInBlock
	}
	// TODO: cache results
	return numL1Messages
}

// MustConvertTxDataToRLPEncoding transforms []*TransactionData into []*types.Transaction.
func MustConvertTxDataToRLPEncoding(txData *types.TransactionData) []byte {
	data, err := hexutil.Decode(txData.Data)
	if err != nil {
		log.Crit("failed to decode txData.Data", "data", txData.Data, "err", err)
	}

	var tx *types.Transaction
	switch txData.Type {
	case types.LegacyTxType:
		tx = types.NewTx(&types.LegacyTx{
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

	case types.AccessListTxType:
		tx = types.NewTx(&types.AccessListTx{
			ChainID:    txData.ChainId.ToInt(),
			Nonce:      txData.Nonce,
			To:         txData.To,
			Value:      txData.Value.ToInt(),
			Gas:        txData.Gas,
			GasPrice:   txData.GasPrice.ToInt(),
			Data:       data,
			AccessList: txData.AccessList,
			V:          txData.V.ToInt(),
			R:          txData.R.ToInt(),
			S:          txData.S.ToInt(),
		})

	case types.DynamicFeeTxType:
		tx = types.NewTx(&types.DynamicFeeTx{
			ChainID:    txData.ChainId.ToInt(),
			Nonce:      txData.Nonce,
			To:         txData.To,
			Value:      txData.Value.ToInt(),
			Gas:        txData.Gas,
			GasTipCap:  txData.GasTipCap.ToInt(),
			GasFeeCap:  txData.GasFeeCap.ToInt(),
			Data:       data,
			AccessList: txData.AccessList,
			V:          txData.V.ToInt(),
			R:          txData.R.ToInt(),
			S:          txData.S.ToInt(),
		})

	case types.L1MessageTxType:
	default:
		log.Crit("unsupported tx type", "type", txData.Type)
	}

	rlpTxData, err := tx.MarshalBinary()
	if err != nil {
		log.Crit("failed to marshal binary of the tx", "tx", tx, "err", err)
	}

	return rlpTxData
}

// GetCrcMax calculates the maximum row consumption of crc.
func (c *Chunk) GetCrcMax() (uint64, error) {
	// Map sub-circuit name to row count
	crc := make(map[string]uint64)

	// Iterate over blocks, accumulate row consumption
	for _, block := range c.Blocks {
		if block.RowConsumption == nil {
			return 0, fmt.Errorf("block (%d, %v) has nil RowConsumption", block.Header.Number, block.Header.Hash().Hex())
		}
		for _, subCircuit := range *block.RowConsumption {
			crc[subCircuit.Name] += subCircuit.RowNumber
		}
	}

	// Find the maximum row consumption
	var maxVal uint64
	for _, value := range crc {
		if value > maxVal {
			maxVal = value
		}
	}

	// Return the maximum row consumption
	return maxVal, nil
}

// GetNumTransactions calculates the total number of transactions in a Chunk.
func (c *Chunk) GetNumTransactions() uint64 {
	var totalTxNum uint64
	for _, block := range c.Blocks {
		totalTxNum += uint64(len(block.Transactions))
	}
	return totalTxNum
}

// GetNumL2Transactions calculates the total number of L2 transactions in a Chunk.
func (c *Chunk) GetNumL2Transactions() uint64 {
	var totalTxNum uint64
	for _, block := range c.Blocks {
		totalTxNum += block.NumL2Transactions()
	}
	return totalTxNum
}

// GetL2GasUsed calculates the total gas of L2 transactions in a Chunk.
func (c *Chunk) GetL2GasUsed() uint64 {
	var totalTxNum uint64
	for _, block := range c.Blocks {
		totalTxNum += block.Header.GasUsed
	}
	return totalTxNum
}

// GetStateRoot gets the state root after committing/finalizing the batch.
func (b *Batch) GetStateRoot() common.Hash {
	numChunks := len(b.Chunks)
	if len(b.Chunks) == 0 {
		return common.Hash{}
	}
	lastChunkBlockNum := len(b.Chunks[numChunks-1].Blocks)
	return b.Chunks[len(b.Chunks)-1].Blocks[lastChunkBlockNum-1].Header.Root
}

// GetWithdrawRoot gets the withdraw root after committing/finalizing the batch.
func (b *Batch) GetWithdrawRoot() common.Hash {
	numChunks := len(b.Chunks)
	if len(b.Chunks) == 0 {
		return common.Hash{}
	}
	lastChunkBlockNum := len(b.Chunks[numChunks-1].Blocks)
	return b.Chunks[len(b.Chunks)-1].Blocks[lastChunkBlockNum-1].WithdrawRoot
}

// GetNumChunks gets the number of chunks of the batch.
func (b *Batch) GetNumChunks() uint64 {
	return uint64(len(b.Chunks))
}
