package types

import (
	"encoding/hex"
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
	hexString := hex.EncodeToString(bytes)
	assert.NoError(t, err)
	assert.Equal(t, 299, len(bytes))
	assert.Equal(t, "0100000000000000020000000063807b2a0000000000000000000000000000000000000000000000000000000000000000000355418d1e81840002000000000073f87180843b9aec2e8307a12094c0c4c8baea3f6acb49b6e1fb9e2adeceeacb0ca28a152d02c7e14af60000008083019ecea0ab07ae99c67aa78e7ba5cf6781e90cc32b219b1de102513d56548a41e86df514a034cbd19feacd73e8ce64d00c4d1996b9b5243c578fd7f51bfaec288bbaf42a8b00000073f87101843b9aec2e8307a1209401bae6bf68e9a03fb2bc0615b1bf0d69ce9411ed8a152d02c7e14af60000008083019ecea0f039985866d8256f10c1be4f7b2cace28d8f20bde27e2604393eb095b7f77316a05a3e6e81065f2b4604bcec5bd4aba684835996fc3f879380aac1c09c6eed32f1", hexString)
}

func TestChunkHash(t *testing.T) {
	// Test case 1: when the chunk contains no blocks
	chunk := &Chunk{
		Blocks: []*WrappedBlock{},
	}
	bytes, err := chunk.Hash()
	assert.Nil(t, bytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number of blocks is 0")

	// Test case 2: successfully hashing a chunk on one block
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
	hexString := hex.EncodeToString(bytes)
	assert.NoError(t, err)
	assert.Equal(t, "78c839dfc494396c16b40946f32b3f4c3e8c2d4bfd04aefcf235edec474482f8", hexString)

	// Test case 3: successfully hashing a chunk on two blocks
	templateBlockTrace1, err := os.ReadFile("../testdata/blockTrace_03.json")
	assert.NoError(t, err)
	wrappedBlock1 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace1, wrappedBlock1))
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock,
			wrappedBlock1,
		},
	}
	bytes, err = chunk.Hash()
	hexString = hex.EncodeToString(bytes)
	assert.NoError(t, err)
	assert.Equal(t, "aa9e494f72bc6965857856f0fae6916f27b2a6591c714a573b2fab46df03b8ae", hexString)
}
