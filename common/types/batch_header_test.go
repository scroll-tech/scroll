package types

import (
	"encoding/json"
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
	assert.Equal(t, 32, len(batchHeader.skippedL1MessageBitmap))
	expectedBitmap := "00000000000000000000000000000000000000000000000000000000000003ff" // skip first 10
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(batchHeader.skippedL1MessageBitmap))

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
	assert.Equal(t, 32, len(batchHeader.skippedL1MessageBitmap))
	expectedBitmap = "0000000000000000000000000000000000000000000000000000000000000000" // all bits are included, so none are skipped
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(batchHeader.skippedL1MessageBitmap))

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
	assert.Equal(t, 32, len(batchHeader.skippedL1MessageBitmap))
	expectedBitmap = "0000000000000000000000000000000000000000000000000000001fffffffff" // skipped the first 37 messages
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(batchHeader.skippedL1MessageBitmap))

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
	assert.Equal(t, 32, len(batchHeader.skippedL1MessageBitmap))
	expectedBitmap = "00000000000000000000000000000000000000000000000000000000000001dd" // 0111011101
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(batchHeader.skippedL1MessageBitmap))

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
	assert.Equal(t, 64, len(batchHeader.skippedL1MessageBitmap))
	expectedBitmap = "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd0000000000000000000000000000000000000000000000000000000000000000"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(batchHeader.skippedL1MessageBitmap))
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
	assert.Equal(t, 129, len(bytes))
	assert.Equal(t, "0100000000000000010000000000000000000000000000000079841093f56d4e454a27371c924b604f9f1831bcecf26ef5549a4b86b5f7cc1b7afdc2ea6f8daaa4b430ce1424f59bcec401d00e34a99b1da457babc405a86070000000000000000290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563", common.Bytes2Hex(bytes))

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
	assert.Equal(t, 161, len(bytes))
	assert.Equal(t, "010000000000000001000000000000000b000000000000000bd66e72c479686e1f25b496c0fa38f8722b3fdd381ce3bf56e78129b510adbbd77afdc2ea6f8daaa4b430ce1424f59bcec401d00e34a99b1da457babc405a860700000000000000000000000000000000000000000000000000000000000003ff0000000000000000290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563", common.Bytes2Hex(bytes))
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
	assert.Equal(t, "c1c81ddb1216d8bcb26d8fb0b60d3c10a3f37c15cdd53893ea31e76b20de51f4", common.Bytes2Hex(hash.Bytes()))

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
	assert.Equal(t, "c2ce574d3331ea9f7a352a0b1fb7e90db246590938c8e7a9b39ff53a23a1a568", common.Bytes2Hex(hash2.Bytes()))

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
	assert.Equal(t, "a66ce437f630868893f2f2fc6894a50363a128a6db1bf25ba0b19a16f1bf5361", common.Bytes2Hex(hash.Bytes()))
}

func TestBatchHeaderDecode(t *testing.T) {
	header := &BatchHeader{
		version:                1,
		batchIndex:             10,
		l1MessagePopped:        20,
		totalL1MessagePopped:   30,
		dataHash:               common.HexToHash("0x01"),
		parentBatchHash:        common.HexToHash("0x02"),
		skippedL1MessageBitmap: []byte{0x01, 0x02, 0x03},
		lastAppliedL1Block:     5,
		l1BlockRangeHash:       common.HexToHash("438ed7f9d8d5a312b5eab7527789c7c1fbb26c9b2700e5f4ce0facd7824bd5ba"),
	}

	encoded := header.Encode()
	decoded, err := DecodeBatchHeader(encoded)
	assert.NoError(t, err)
	assert.Equal(t, header, decoded)
}
