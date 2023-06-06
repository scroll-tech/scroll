package types

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestNewBatchHeader(t *testing.T) {
	templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_02.json")
	assert.NoError(t, err)

	wrappedBlock := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock))
	chunk := &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock,
		},
	}
	parentBatchHeader := &BatchHeader{
		lastBatchQueueIndex:    0,
		version:                1,
		batchIndex:             0,
		l1MessagePopped:        0,
		totalL1MessagePopped:   0,
		dataHash:               common.HexToHash("0x0"),
		parentBatchHash:        common.HexToHash("0x0"),
		skippedL1MessageBitmap: nil,
	}
	batchHeader, err := NewBatchHeader(1, 1, 0, parentBatchHeader, []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
}

func TestBatchHeaderEncode(t *testing.T) {
	templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_02.json")
	assert.NoError(t, err)

	wrappedBlock := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock))
	chunk := &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock,
		},
	}
	parentBatchHeader := &BatchHeader{
		lastBatchQueueIndex:    0,
		version:                1,
		batchIndex:             0,
		l1MessagePopped:        0,
		totalL1MessagePopped:   0,
		dataHash:               common.HexToHash("0x0"),
		parentBatchHash:        common.HexToHash("0x0"),
		skippedL1MessageBitmap: nil,
	}
	batchHeader, err := NewBatchHeader(1, 1, 0, parentBatchHeader, []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	bytes := batchHeader.Encode()
	assert.Equal(t, 89, len(bytes))
	assert.Equal(t, "01000000000000000100000000000000000000000000000000a8d9704a9432e7e16433d2d90026e98db8feb23e0c02c2517b0bd27ef851d2f34136709aabc8a23aa17fbcc833da2f7857d3c2884feec9aae73429c135f94985", common.Bytes2Hex(bytes))
}

func TestBatchHeaderHash(t *testing.T) {
	templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_02.json")
	assert.NoError(t, err)

	wrappedBlock := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock))
	chunk := &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock,
		},
	}
	parentBatchHeader := &BatchHeader{
		lastBatchQueueIndex:    0,
		version:                1,
		batchIndex:             0,
		l1MessagePopped:        0,
		totalL1MessagePopped:   0,
		dataHash:               common.HexToHash("0x0"),
		parentBatchHash:        common.HexToHash("0x0"),
		skippedL1MessageBitmap: nil,
	}
	batchHeader, err := NewBatchHeader(1, 1, 0, parentBatchHeader, []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	hash := batchHeader.Hash()
	assert.Equal(t, "01a66caaa1ed11e73a652c25f6fb82f5751746d8782dab76220f0c2f3b07c662", common.Bytes2Hex(hash.Bytes()))

	templateBlockTrace, err = os.ReadFile("../testdata/blockTrace_03.json")
	assert.NoError(t, err)

	wrappedBlock2 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock2))
	chunk2 := &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock2,
		},
	}
	batchHeader2, err := NewBatchHeader(1, 2, 0, batchHeader, []*Chunk{chunk2})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader2)
	hash2 := batchHeader2.Hash()
	assert.Equal(t, "b913812b58a6aa49e058dcf2a4734888c7b4a5832e04aae3a5aeca6d3c74669f", common.Bytes2Hex(hash2.Bytes()))
}
