package types

import (
	"encoding/binary"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
)

type BatchHeader struct {
	// Public input hash only (finalization)
	PrevStateRoot    common.Hash
	NewStateRoot     common.Hash
	WithdrawTrieRoot common.Hash

	lastBatchQueueIndex uint64 // last queue index from the previous batch (last chunk)

	// Encoded in BatchHeaderV0Codec
	version                uint8
	batchIndex             uint64
	l1MessagePopped        uint64
	totalL1MessagePopped   uint64
	dataHash               common.Hash
	parentBatchHash        common.Hash
	skippedL1MessageBitmap []*big.Int // LSB is the first L1 message
}

func NewBatchHeader(version uint8, batchIndex, totalL1MessagePoppedBefore uint64, parentBatchHeader *BatchHeader, chunks []*Chunk) (*BatchHeader, error) {
	// calculate `l1MessagePopped`, `totalL1MessagePopped`, `dataHash`, and `skippedL1MessageBitmap` based on `chunks`
	txBytes := make([]byte, 0)
	blockBytes := make([]byte, 0)
	var l1MessagePopped uint64 = 0
	var newLastBatchQueueIndex uint64
	skippedL1MessageBitmap := make([]*big.Int, 0)

	lastBatchQueueIndex := parentBatchHeader.lastBatchQueueIndex
	firstTx := true

	for i, chunk := range chunks {

		// intialize uint256 bitmap to all 0s, 1s mean skip, 0s mean pop
		bitmap := big.NewInt(0)
		// bitmap index offset
		var bitmapIndexOffset uint64
		// starting queue index for chunk
		var lastChunkQueueIndex uint64
		if firstTx {
			firstTx = false
			lastChunkQueueIndex = lastBatchQueueIndex
		}

		//Build l1MessagePopped
		for j, block := range chunk.Blocks {
			for k, tx := range block.Transactions {
				if tx.Type != 0x7E {
					continue
				}

				queueIndex := tx.Nonce
				bitmapIndexOffset = queueIndex - lastChunkQueueIndex

				// Check if offset exceeds 256, exit the block
				if bitmapIndexOffset > 255 {
					// TODO
					break
				}

				bitmap.Or(bitmap, new(big.Int).Lsh(big.NewInt(1), uint(bitmapIndexOffset)))
				l1MessagePopped += 1

				// new lastBatchQueueIndex
				if j == len(chunk.Blocks)-1 && k == len(block.Transactions)-1 {
					if i == len(chunks)-1 {
						newLastBatchQueueIndex = queueIndex
					} else { // new lastChunkQueueIndex
						lastChunkQueueIndex = queueIndex
					}
				}
			}
		}
		bitmapReversed := big.NewInt(0)
		// Reverse the bits in bitmap.
		for i := 0; i < bitmap.BitLen(); i++ {
			// Get the bit at index i.
			bit := bitmap.Bit(i)

			// Set the bit at the index equal to 256 - i - 1 in bitmapReversed to bit.
			bitmapReversed.SetBit(bitmapReversed, 256-i-1, bit)
		}

		skippedL1MessageBitmap = append(skippedL1MessageBitmap, bitmapReversed)

		// Build dataHash
		chunkCodec, err := chunk.Encode()

		if err != nil {
			return nil, err
		}

		numBlocks := chunkCodec[0]

		// concatenate block contexts
		blockBytes = append(blockBytes, chunkCodec[1:60*numBlocks+1]...)

		// retrieve block context start index
		blockIndex := 1

		// retrieve l2 tx start index
		l2TxIndex := uint32(60*numBlocks + 1)

		// l2 tx hashes
		l2TxHashes := make([]byte, 0)

		for numBlocks > 0 {
			// TODO: concatenate l1 message hashes

			// concatenate l2 txs hashes
			// retrieve the number of transactions in current block.
			numTransactionsIndex := blockIndex + 56
			numTxsInBlock := binary.BigEndian.Uint16(chunkCodec[numTransactionsIndex : numTransactionsIndex+2])

			for numTxsInBlock > 0 {
				l2TxLen := binary.BigEndian.Uint32(chunkCodec[l2TxIndex : l2TxIndex+4])
				l2TxIndex += 4
				txPayload := chunkCodec[l2TxIndex : l2TxIndex+l2TxLen]
				txHash := crypto.Keccak256Hash(txPayload).Bytes()
				l2TxIndex += l2TxLen

				l2TxHashes = append(l2TxHashes, txHash...)
				numTxsInBlock--
			}

			numBlocks--
			blockIndex += 60
		}
		txBytes = append(txBytes, l2TxHashes...)
	}
	// hash data
	dataHash := crypto.Keccak256Hash(blockBytes, txBytes)

	return &BatchHeader{
		lastBatchQueueIndex:    newLastBatchQueueIndex,
		version:                version,
		batchIndex:             batchIndex,
		l1MessagePopped:        0,
		totalL1MessagePopped:   totalL1MessagePoppedBefore + l1MessagePopped,
		dataHash:               dataHash,
		parentBatchHash:        parentBatchHeader.Hash(),
		skippedL1MessageBitmap: skippedL1MessageBitmap,
	}, nil
}

// encode batchHeader
// reference: https://github.com/scroll-tech/scroll/blob/develop/contracts/src/libraries/codec/BatchHeaderV0Codec.sol#L5
func (b *BatchHeader) Encode() []byte {
	batchBytes := make([]byte, 0)

	batchBytes = append(batchBytes, byte(b.version))

	batchIndexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(batchIndexBytes, b.batchIndex)
	batchBytes = append(batchBytes, batchIndexBytes...)

	l1MessagePoppedBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(l1MessagePoppedBytes, b.l1MessagePopped)
	batchBytes = append(batchBytes, l1MessagePoppedBytes...)

	totalL1MessagePoppedBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(totalL1MessagePoppedBytes, b.totalL1MessagePopped)
	batchBytes = append(batchBytes, totalL1MessagePoppedBytes...)

	batchBytes = append(batchBytes, b.dataHash[:]...)

	batchBytes = append(batchBytes, b.parentBatchHash[:]...)

	if b.skippedL1MessageBitmap != nil {
		for _, num := range b.skippedL1MessageBitmap {
			numBytes := num.Bytes()

			// Big Endian padding
			if len(numBytes) < 32 {
				padding := make([]byte, 32-len(numBytes))
				numBytes = append(padding, numBytes...)
			}

			batchBytes = append(batchBytes, numBytes...)
		}
	}

	return batchBytes
}

// calculate batchHeader hash
// reference: https://github.com/scroll-tech/scroll/blob/develop/contracts/src/L1/rollup/ScrollChain.sol#L394
func (b *BatchHeader) Hash() common.Hash {
	return crypto.Keccak256Hash(b.Encode())
}
