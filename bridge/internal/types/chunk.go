package types

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
)

// Chunk contains blocks to be encoded
type Chunk struct {
	Blocks []*WrappedBlock `json:"blocks"`
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

// Encode encodes the Chunk into RollupV2 Chunk Encoding.
func (c *Chunk) Encode(totalL1MessagePoppedBefore uint64) ([]byte, error) {
	numBlocks := len(c.Blocks)

	if numBlocks > 255 {
		return nil, errors.New("number of blocks exceeds 1 byte")
	}
	if numBlocks == 0 {
		return nil, errors.New("number of blocks is 0")
	}

	var chunkBytes []byte
	chunkBytes = append(chunkBytes, byte(numBlocks))

	var l2TxDataBytes []byte

	for _, block := range c.Blocks {
		blockBytes, err := block.Encode(totalL1MessagePoppedBefore)
		if err != nil {
			return nil, fmt.Errorf("failed to encode block: %v", err)
		}
		totalL1MessagePoppedBefore += block.NumL1Messages(totalL1MessagePoppedBefore)

		if len(blockBytes) != 60 {
			return nil, fmt.Errorf("block encoding is not 60 bytes long %x", len(blockBytes))
		}

		chunkBytes = append(chunkBytes, blockBytes...)

		// Append rlp-encoded l2Txs
		for _, txData := range block.Transactions {
			if txData.Type == L1MessageTxType {
				continue
			}
			data, _ := hexutil.Decode(txData.Data)
			// right now we only support legacy tx
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
			var txLen [4]byte
			binary.BigEndian.PutUint32(txLen[:], uint32(len(rlpTxData)))
			l2TxDataBytes = append(l2TxDataBytes, txLen[:]...)
			l2TxDataBytes = append(l2TxDataBytes, rlpTxData...)
		}
	}

	chunkBytes = append(chunkBytes, l2TxDataBytes...)

	return chunkBytes, nil
}

// Hash hashes the Chunk into RollupV2 Chunk Hash
func (c *Chunk) Hash(totalL1MessagePoppedBefore uint64) (common.Hash, error) {
	chunkBytes, err := c.Encode(totalL1MessagePoppedBefore)
	if err != nil {
		return common.Hash{}, err
	}
	numBlocks := chunkBytes[0]

	// concatenate block contexts
	var dataBytes []byte
	for i := 0; i < int(numBlocks); i++ {
		// only first 58 bytes is needed
		dataBytes = append(dataBytes, chunkBytes[1+60*i:60*i+59]...)
	}

	// concatenate l1 and l2 tx hashes
	for _, block := range c.Blocks {
		var l1TxHashes []byte
		var l2TxHashes []byte
		for _, txData := range block.Transactions {
			txHash := strings.TrimPrefix(txData.TxHash, "0x")
			hashBytes, err := hex.DecodeString(txHash)
			if err != nil {
				return common.Hash{}, err
			}
			if txData.Type == L1MessageTxType {
				l1TxHashes = append(l1TxHashes, hashBytes...)
			} else {
				l2TxHashes = append(l2TxHashes, hashBytes...)
			}
		}
		dataBytes = append(dataBytes, l1TxHashes...)
		dataBytes = append(dataBytes, l2TxHashes...)
	}

	hash := crypto.Keccak256Hash(dataBytes)
	return hash, nil
}
