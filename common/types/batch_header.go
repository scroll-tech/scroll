package types

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
)

// BatchMeta contains metadata of a batch.
type BatchMeta struct {
	StartChunkIndex           uint64
	StartChunkHash            string
	EndChunkIndex             uint64
	EndChunkHash              string
	TotalL1CommitGas          uint64
	TotalL1CommitCalldataSize uint32
}

// BatchHeader contains batch header info to be committed.
type BatchHeader struct {
	// Encoded in BatchHeaderV0Codec
	version                uint8
	batchIndex             uint64
	l1MessagePopped        uint64
	totalL1MessagePopped   uint64
	dataHash               common.Hash
	parentBatchHash        common.Hash
	skippedL1MessageBitmap []byte
	lastAppliedL1Block     uint64
	l1BlockRangeHash       common.Hash
}

// NewBatchHeader creates a new BatchHeader
func NewBatchHeader(version uint8, batchIndex, totalL1MessagePoppedBefore uint64, parentBatchHash common.Hash, chunks []*Chunk) (*BatchHeader, error) {
	// buffer for storing chunk hashes in order to compute the batch data hash
	var dataBytes []byte

	// skipped L1 message bitmap, an array of 256-bit bitmaps
	var skippedBitmap []*big.Int

	// the first queue index that belongs to this batch
	baseIndex := totalL1MessagePoppedBefore

	// the next queue index that we need to process
	nextIndex := totalL1MessagePoppedBefore

	for chunkID, chunk := range chunks {
		// build data hash
		totalL1MessagePoppedBeforeChunk := nextIndex
		chunkHash, err := chunk.Hash(totalL1MessagePoppedBeforeChunk)
		if err != nil {
			return nil, err
		}
		dataBytes = append(dataBytes, chunkHash.Bytes()...)

		// build skip bitmap
		for blockID, block := range chunk.Blocks {
			for _, tx := range block.Transactions {
				if tx.Type != types.L1MessageTxType {
					continue
				}
				currentIndex := tx.Nonce

				if currentIndex < nextIndex {
					return nil, fmt.Errorf("unexpected batch payload, expected queue index: %d, got: %d. Batch index: %d, chunk index in batch: %d, block index in chunk: %d, block hash: %v, transaction hash: %v", nextIndex, currentIndex, batchIndex, chunkID, blockID, block.Header.Hash(), tx.TxHash)
				}

				// mark skipped messages
				for skippedIndex := nextIndex; skippedIndex < currentIndex; skippedIndex++ {
					quo := int((skippedIndex - baseIndex) / 256)
					rem := int((skippedIndex - baseIndex) % 256)
					for len(skippedBitmap) <= quo {
						bitmap := big.NewInt(0)
						skippedBitmap = append(skippedBitmap, bitmap)
					}
					skippedBitmap[quo].SetBit(skippedBitmap[quo], rem, 1)
				}

				// process included message
				quo := int((currentIndex - baseIndex) / 256)
				for len(skippedBitmap) <= quo {
					bitmap := big.NewInt(0)
					skippedBitmap = append(skippedBitmap, bitmap)
				}

				nextIndex = currentIndex + 1
			}
		}
	}

	// compute data hash
	dataHash := crypto.Keccak256Hash(dataBytes)

	// compute skipped bitmap
	bitmapBytes := make([]byte, len(skippedBitmap)*32)
	for ii, num := range skippedBitmap {
		bytes := num.Bytes()
		padding := 32 - len(bytes)
		copy(bitmapBytes[32*ii+padding:], bytes)
	}

	return &BatchHeader{
		version:                version,
		batchIndex:             batchIndex,
		l1MessagePopped:        nextIndex - totalL1MessagePoppedBefore,
		totalL1MessagePopped:   nextIndex,
		dataHash:               dataHash,
		parentBatchHash:        parentBatchHash,
		skippedL1MessageBitmap: bitmapBytes,
		// TODO:
		lastAppliedL1Block: 0,
		l1BlockRangeHash:   common.Hash{},
	}, nil
}

// Version returns the version of the BatchHeader.
func (b *BatchHeader) Version() uint8 {
	return b.version
}

// BatchIndex returns the batch index of the BatchHeader.
func (b *BatchHeader) BatchIndex() uint64 {
	return b.batchIndex
}

// TotalL1MessagePopped returns the total number of L1 messages popped in the BatchHeader.
func (b *BatchHeader) TotalL1MessagePopped() uint64 {
	return b.totalL1MessagePopped
}

// SkippedL1MessageBitmap returns the skipped L1 message bitmap in the BatchHeader.
func (b *BatchHeader) SkippedL1MessageBitmap() []byte {
	return b.skippedL1MessageBitmap
}

// Encode encodes the BatchHeader into RollupV2 BatchHeaderV0Codec Encoding.
func (b *BatchHeader) Encode() []byte {
	batchBytes := make([]byte, 129+len(b.skippedL1MessageBitmap))
	batchBytes[0] = b.version
	binary.BigEndian.PutUint64(batchBytes[1:], b.batchIndex)
	binary.BigEndian.PutUint64(batchBytes[9:], b.l1MessagePopped)
	binary.BigEndian.PutUint64(batchBytes[17:], b.totalL1MessagePopped)
	copy(batchBytes[25:], b.dataHash[:])
	copy(batchBytes[57:], b.parentBatchHash[:])
	copy(batchBytes[57:], b.parentBatchHash[:])
	copy(batchBytes[89:], b.skippedL1MessageBitmap[:])
	binary.BigEndian.PutUint64(batchBytes[89+len(b.skippedL1MessageBitmap):], b.lastAppliedL1Block)
	copy(batchBytes[97+len(b.skippedL1MessageBitmap):], b.l1BlockRangeHash[:])
	return batchBytes
}

// Hash calculates the hash of the batch header.
func (b *BatchHeader) Hash() common.Hash {
	return crypto.Keccak256Hash(b.Encode())
}

// DecodeBatchHeader attempts to decode the given byte slice into a BatchHeader.
func DecodeBatchHeader(data []byte) (*BatchHeader, error) {
	if len(data) < 97 {
		return nil, fmt.Errorf("insufficient data for BatchHeader")
	}
	b := &BatchHeader{
		version:                data[0],
		batchIndex:             binary.BigEndian.Uint64(data[1:9]),
		l1MessagePopped:        binary.BigEndian.Uint64(data[9:17]),
		totalL1MessagePopped:   binary.BigEndian.Uint64(data[17:25]),
		dataHash:               common.BytesToHash(data[25:57]),
		parentBatchHash:        common.BytesToHash(data[57:89]),
		skippedL1MessageBitmap: data[89 : len(data)-40],
		lastAppliedL1Block:     binary.BigEndian.Uint64(data[len(data)-40 : len(data)-32]),
		l1BlockRangeHash:       common.BytesToHash(data[len(data)-32:]),
	}
	return b, nil
}
