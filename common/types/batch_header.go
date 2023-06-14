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
	var l1MessagePopped uint64
	var skippedL1MessageBitmap []*big.Int

	// Starting queue index for whole batch
	lastBatchIndex := totalL1MessagePoppedBefore + 1
	// Previous queue index which popped
	lastPoppedIndex := totalL1MessagePoppedBefore
	for _, chunk := range chunks {
		for _, block := range chunk.Blocks {
			for _, tx := range block.Transactions {
				if tx.Type != 0x7E {
					continue
				}

				for skippedMsg := lastPoppedIndex + 1; skippedMsg < tx.Nonce; skippedMsg++ {
					offset := int((skippedMsg - lastBatchIndex) / 256)
					for offset >= len(skippedL1MessageBitmap) {
						bitmap := big.NewInt(0)
						skippedL1MessageBitmap = append(skippedL1MessageBitmap, bitmap)
					}
					rem := int((skippedMsg - lastBatchIndex) % 256)
					skippedL1MessageBitmap[offset].SetBit(skippedL1MessageBitmap[offset], rem, 1)
				}
				lastPoppedIndex = tx.Nonce
				l1MessagePopped++
				offset := int(lastPoppedIndex-lastBatchIndex) / 256
				for offset >= len(skippedL1MessageBitmap) {
					bitmap := big.NewInt(0)
					skippedL1MessageBitmap = append(skippedL1MessageBitmap, bitmap)
				}
			}
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
