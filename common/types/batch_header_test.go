package types

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestNewBatchHeader(t *testing.T) {
	// Without L1 Msg
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
		version:                1,
		batchIndex:             0,
		l1MessagePopped:        0,
		totalL1MessagePopped:   0,
		dataHash:               common.HexToHash("0x0"),
		parentBatchHash:        common.HexToHash("0x0"),
		skippedL1MessageBitmap: nil,
	}
	batchHeader, err := NewBatchHeader(1, 1, 0, parentBatchHeader.Hash(), []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	assert.Equal(t, 0, len(batchHeader.skippedL1MessageBitmap))

	// 1 L1 Msg in 1 bitmap
	templateBlockTrace2, err := os.ReadFile("../testdata/blockTrace_04.json")
	assert.NoError(t, err)

	wrappedBlock2 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace2, wrappedBlock2))
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock2,
		},
	}
	batchHeader, err = NewBatchHeader(1, 1, 0, parentBatchHeader.Hash(), []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	assert.Equal(t, 1, len(batchHeader.skippedL1MessageBitmap))

	// many consecutive L1 Msgs in 1 bitmap, no leading skipped msgs
	templateBlockTrace3, err := os.ReadFile("../testdata/blockTrace_05.json")
	assert.NoError(t, err)

	wrappedBlock3 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace3, wrappedBlock3))
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock3,
		},
	}
	batchHeader, err = NewBatchHeader(1, 1, 37, parentBatchHeader.Hash(), []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	assert.Equal(t, uint64(5), batchHeader.l1MessagePopped)
	assert.Equal(t, 1, len(batchHeader.skippedL1MessageBitmap))
	expectedBitmap := big.NewInt(0) // all bits are popped, so none are skipped
	assert.Equal(t, 0, batchHeader.skippedL1MessageBitmap[0].Cmp(expectedBitmap))

	// many consecutive L1 Msgs in 1 bitmap, with leading skipped msgs
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock3,
		},
	}
	batchHeader, err = NewBatchHeader(1, 1, 0, parentBatchHeader.Hash(), []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	assert.Equal(t, uint64(42), batchHeader.l1MessagePopped)
	assert.Equal(t, 1, len(batchHeader.skippedL1MessageBitmap))
	expectedBitmap = new(big.Int)
	expectedBitmap.SetString("000001111111111111111111111111111111111111", 2)
	assert.Equal(t, 0, batchHeader.skippedL1MessageBitmap[0].Cmp(expectedBitmap))

	// many sparse L1 Msgs in 1 bitmap
	templateBlockTrace4, err := os.ReadFile("../testdata/blockTrace_06.json")
	assert.NoError(t, err)

	wrappedBlock4 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace4, wrappedBlock4))
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock4,
		},
	}
	batchHeader, err = NewBatchHeader(1, 1, 0, parentBatchHeader.Hash(), []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	assert.Equal(t, uint64(10), batchHeader.l1MessagePopped)
	assert.Equal(t, 1, len(batchHeader.skippedL1MessageBitmap))
	expectedBitmap = new(big.Int)
	expectedBitmap.SetString("0111011101", 2)
	assert.Equal(t, 0, batchHeader.skippedL1MessageBitmap[0].Cmp(expectedBitmap))

	// many L1 Msgs in each of 2 bitmaps
	templateBlockTrace5, err := os.ReadFile("../testdata/blockTrace_07.json")
	assert.NoError(t, err)

	wrappedBlock5 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace5, wrappedBlock5))
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock5,
		},
	}
	batchHeader, err = NewBatchHeader(1, 1, 0, parentBatchHeader.Hash(), []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	assert.Equal(t, uint64(257), batchHeader.l1MessagePopped)
	assert.Equal(t, 2, len(batchHeader.skippedL1MessageBitmap))
	expectedBitmap = new(big.Int)
	expectedBitmap.SetString("fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", 16)
	assert.Equal(t, 0, batchHeader.skippedL1MessageBitmap[0].Cmp(expectedBitmap))
	expectedBitmap = big.NewInt(0) // all bits are popped, so none are skipped
	assert.Equal(t, 0, batchHeader.skippedL1MessageBitmap[1].Cmp(expectedBitmap))
}

func TestBatchHeaderEncode(t *testing.T) {
	// Without L1 Msg
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
		version:                1,
		batchIndex:             0,
		l1MessagePopped:        0,
		totalL1MessagePopped:   0,
		dataHash:               common.HexToHash("0x0"),
		parentBatchHash:        common.HexToHash("0x0"),
		skippedL1MessageBitmap: nil,
	}
	batchHeader, err := NewBatchHeader(1, 1, 0, parentBatchHeader.Hash(), []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	bytes := batchHeader.Encode()
	assert.Equal(t, 89, len(bytes))
	assert.Equal(t, "0100000000000000010000000000000000000000000000000010a64c9bd905f8caf5d668fbda622d6558c5a42cdb4b3895709743d159c22e534136709aabc8a23aa17fbcc833da2f7857d3c2884feec9aae73429c135f94985", common.Bytes2Hex(bytes))

	// With L1 Msg
	templateBlockTrace2, err := os.ReadFile("../testdata/blockTrace_04.json")
	assert.NoError(t, err)

	wrappedBlock2 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace2, wrappedBlock2))
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock2,
		},
	}
	batchHeader, err = NewBatchHeader(1, 1, 0, parentBatchHeader.Hash(), []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	bytes = batchHeader.Encode()
	assert.Equal(t, 121, len(bytes))
	assert.Equal(t, "01000000000000000100000000000000010000000000000001457a9e90e8e51ba2de2f66c6b589540b88cf594dac7fa7d04b99cdcfecf24e384136709aabc8a23aa17fbcc833da2f7857d3c2884feec9aae73429c135f9498500000000000000000000000000000000000000000000000000000000000001ff", common.Bytes2Hex(bytes))
}

func TestBatchHeaderHash(t *testing.T) {
	// Without L1 Msg
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
		version:                1,
		batchIndex:             0,
		l1MessagePopped:        0,
		totalL1MessagePopped:   0,
		dataHash:               common.HexToHash("0x0"),
		parentBatchHash:        common.HexToHash("0x0"),
		skippedL1MessageBitmap: nil,
	}
	batchHeader, err := NewBatchHeader(1, 1, 0, parentBatchHeader.Hash(), []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	hash := batchHeader.Hash()
	assert.Equal(t, "d69da4357da0073f4093c76e49f077e21bb52f48f57ee3e1fbd9c38a2881af81", common.Bytes2Hex(hash.Bytes()))

	templateBlockTrace, err = os.ReadFile("../testdata/blockTrace_03.json")
	assert.NoError(t, err)

	wrappedBlock2 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock2))
	chunk2 := &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock2,
		},
	}
	batchHeader2, err := NewBatchHeader(1, 2, 0, batchHeader.Hash(), []*Chunk{chunk2})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader2)
	hash2 := batchHeader2.Hash()
	assert.Equal(t, "34de600163aa745d4513113137a5b54960d13f0d3f2849e490c4b875028bf930", common.Bytes2Hex(hash2.Bytes()))

	// With L1 Msg
	templateBlockTrace3, err := os.ReadFile("../testdata/blockTrace_04.json")
	assert.NoError(t, err)

	wrappedBlock3 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace3, wrappedBlock3))
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock3,
		},
	}
	batchHeader, err = NewBatchHeader(1, 1, 0, parentBatchHeader.Hash(), []*Chunk{chunk})
	assert.NoError(t, err)
	assert.NotNil(t, batchHeader)
	hash = batchHeader.Hash()
	assert.Equal(t, "260f91c0e3f285a3c2c93ab233552b3cb372ed8e5b6b3a3603112ea6c5a1a9ee", common.Bytes2Hex(hash.Bytes()))
}
