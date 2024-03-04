package encoding

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

type Block struct {
	Header       *types.Header
	Transactions []*types.TransactionData
}

type Chunk struct {
	Blocks []*Block `json:"blocks"`
}

type Batch struct {
	Index                      uint64
	TotalL1MessagePoppedBefore uint64
	ParentBatchHash            common.Hash
	Chunks                     []*Chunk
}

// NumL1Messages returns the number of L1 messages in this block.
// This number is the sum of included and skipped L1 messages.
func (w *Block) NumL1Messages(totalL1MessagePoppedBefore uint64) uint64 {
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
func (w *Block) NumL2Transactions() uint64 {
	var count uint64
	for _, txData := range w.Transactions {
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

func ConvertTxDataToRLPEncoding(txData *types.TransactionData) ([]byte, error) {
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
