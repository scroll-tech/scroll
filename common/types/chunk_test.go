package types

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunkEncode(t *testing.T) {
	// Test case 1: when the chunk contains no blocks.
	chunk := &Chunk{
		Blocks: []*WrappedBlock{},
	}
	bytes, err := chunk.Encode()
	assert.Nil(t, bytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number of blocks is 0")

	// Test case 2: when the chunk contains more than 255 blocks.
	chunk = &Chunk{
		Blocks: []*WrappedBlock{},
	}
	for i := 0; i < 256; i++ {
		chunk.Blocks = append(chunk.Blocks, &WrappedBlock{})
	}
	bytes, err = chunk.Encode()
	assert.Nil(t, bytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number of blocks exceeds 1 byte")

	// Test case 3: when the chunk contains one block.
	templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_02.json")
	assert.NoError(t, err)

	wrappedBlock := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock))
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock,
		},
	}
	bytes, err = chunk.Encode()
	assert.NoError(t, err)
	assert.Equal(t, 299, len(bytes))
}

func TestChunkHash(t *testing.T) {
		// Test case 1: when the chunk contains no blocks
		chunk := &Chunk{
			Blocks: []*WrappedBlock{},
		}
		bytes, err := chunk.Hash();
		assert.Nil(t, bytes)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "number of blocks is 0")

		// Test case 2: successfully hashing a chunk
		templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_02.json")
		assert.NoError(t, err)

		wrappedBlock := &WrappedBlock{}
		assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock))
		chunk = &Chunk{
			Blocks: []*WrappedBlock{
				wrappedBlock,
			},
		}
		bytes, err = chunk.Hash()
		assert.NoError(t, err)
		assert.Equal(t, "0x8f1447573740b3e75b979879866b8ad02eecf88e1946275eb8cf14ab95876efc", bytes)
}