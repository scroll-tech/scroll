package types

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/scroll-tech/go-ethereum/common"
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
			if txData.Type == types.L1MessageTxType {
				continue
			}
			rlpTxData, err := convertTxDataToRLPEncoding(txData)
			if err != nil {
				return nil, err
			}
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
		// only the first 58 bytes of each BlockContext are needed for the hashing process
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
			if txData.Type == types.L1MessageTxType {
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

// EstimateL1CommitGas calculates the total L1 commit gas for this chunk approximately
func (c *Chunk) EstimateL1CommitGas() uint64 {
	var totalTxNum uint64
	var totalL1CommitGas uint64
	for _, block := range c.Blocks {
		totalTxNum += uint64(len(block.Transactions))
		totalL1CommitGas += block.EstimateL1CommitGas()
	}

	numBlocks := uint64(len(c.Blocks))
	totalL1CommitGas += 100 * numBlocks                         // numBlocks times warm sload
	totalL1CommitGas += CalldataNonZeroByteGas                  // numBlocks field of chunk encoding in calldata
	totalL1CommitGas += CalldataNonZeroByteGas * numBlocks * 60 // numBlocks of BlockContext in chunk

	totalL1CommitGas += GetKeccak256Gas(58*numBlocks + 32*totalTxNum) // chunk hash
	return totalL1CommitGas
}
