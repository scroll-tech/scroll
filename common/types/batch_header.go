package types

import (
	"encoding/binary"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
)

// BatchHeader contains batch header info to be committed.
type BatchHeader struct {
	// Encoded in BatchHeaderV0Codec
	version                uint8
	batchIndex             uint64
	l1MessagePopped        uint64
	totalL1MessagePopped   uint64
	dataHash               common.Hash
	parentBatchHash        common.Hash
	skippedL1MessageBitmap []*big.Int // LSB is the first L1 message
}

// NewBatchHeader creates a new BatchHeader
func NewBatchHeader(version uint8, batchIndex, totalL1MessagePoppedBefore uint64, parentBatchHash common.Hash, chunks []*Chunk) (*BatchHeader, error) {
	var dataBytes []byte
	var l1MessagePopped uint64 = 0
	var skippedL1MessageBitmap []*big.Int

	// previous queue index which represents a bitmap border
	lastBitmapIndex := totalL1MessagePoppedBefore
	// bitmap index offset
	var bitmapIndexOffset uint64
	// initialize uint256 bitmap
	bitmap := big.NewInt(0)
	for _, chunk := range chunks {
		for _, block := range chunk.Blocks {
			for _, tx := range block.Transactions {
				if tx.Type != 0x7E {
					continue
				}

				queueIndex := tx.Nonce
				bitmapIndexOffset = queueIndex - lastBitmapIndex - 1
				newSize := int(bitmapIndexOffset / 255)
				newBitmaps := newSize - len(skippedL1MessageBitmap)

				// Check if offset exceeds 256 msgs, create new bitmap
				if newBitmaps > 0 {
					for newBitmaps > 0 {
						flippedBitmap := new(big.Int)
						for i := 0; i < 256; i++ {
							bit := bitmap.Bit(i)
							flippedBit := bit ^ 1
							flippedBitmap.SetBit(flippedBitmap, i, flippedBit)
						}
						skippedL1MessageBitmap = append(skippedL1MessageBitmap, flippedBitmap)
						bitmap = big.NewInt(0) // reinitialize bitmap
						newBitmaps--
					}
					// account for the skipped msgs in the new bitmap
					bitmapIndexOffset = bitmapIndexOffset - 255
					lastBitmapIndex = queueIndex - bitmapIndexOffset
				}

				bitmap.SetBit(bitmap, int(bitmapIndexOffset), 1)
				l1MessagePopped++
			}
		}

		// edge case: if skippedL1MessageBitmap length is 0 and bitmap is 0,
		// then we don't need to append skippedL1MessageBitmap
		if len(skippedL1MessageBitmap) != 0 || bitmap.BitLen() != 0 {
			// Append last bitmap
			flippedBitmap := new(big.Int)
			// Flipping up to the last popped index, leaving remaining bits as 0
			for i := 0; i < bitmap.BitLen(); i++ {
				bit := bitmap.Bit(i)
				flippedBit := bit ^ 1
				flippedBitmap.SetBit(flippedBitmap, i, flippedBit)
			}
			skippedL1MessageBitmap = append(skippedL1MessageBitmap, flippedBitmap)
		}

		// Build dataHash
		chunkBytes, err := chunk.Hash()
		if err != nil {
			return nil, err
		}
		dataBytes = append(dataBytes, chunkBytes...)
	}
	dataHash := crypto.Keccak256Hash(dataBytes)

	return &BatchHeader{
		version:                version,
		batchIndex:             batchIndex,
		l1MessagePopped:        l1MessagePopped,
		totalL1MessagePopped:   totalL1MessagePoppedBefore + l1MessagePopped,
		dataHash:               dataHash,
		parentBatchHash:        parentBatchHash,
		skippedL1MessageBitmap: skippedL1MessageBitmap,
	}, nil
}

// Encode encodes the BatchHeader into RollupV2 BatchHeaderV0Codec Encoding.
func (b *BatchHeader) Encode() []byte {
	batchBytes := make([]byte, 89)
	batchBytes[0] = b.version
	binary.BigEndian.PutUint64(batchBytes[1:], b.batchIndex)
	binary.BigEndian.PutUint64(batchBytes[9:], b.l1MessagePopped)
	binary.BigEndian.PutUint64(batchBytes[17:], b.totalL1MessagePopped)
	copy(batchBytes[25:], b.dataHash[:])
	copy(batchBytes[57:], b.parentBatchHash[:])
	for _, num := range b.skippedL1MessageBitmap {
		numBytes := num.Bytes()
		// Big Endian padding
		if len(numBytes) < 32 {
			padding := make([]byte, 32-len(numBytes))
			numBytes = append(padding, numBytes...)
		}
		batchBytes = append(batchBytes, numBytes...)
	}

	return batchBytes
}

// Hash calculates the hash of the batch header.
func (b *BatchHeader) Hash() common.Hash {
	return crypto.Keccak256Hash(b.Encode())
}
