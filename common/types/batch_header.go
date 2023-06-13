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
	// TODO calculate `l1MessagePopped`, `totalL1MessagePopped`, and `skippedL1MessageBitmap` based on `chunks`
	var dataBytes []byte
	for _, chunk := range chunks {
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
		l1MessagePopped:        0,                          // TODO
		totalL1MessagePopped:   totalL1MessagePoppedBefore, // TODO
		dataHash:               dataHash,
		parentBatchHash:        parentBatchHash,
		skippedL1MessageBitmap: nil, // TODO
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
	// TODO: encode skippedL1MessageBitmap

	return batchBytes
}

// Hash calculates the hash of the batch header.
func (b *BatchHeader) Hash() common.Hash {
	return crypto.Keccak256Hash(b.Encode())
}
