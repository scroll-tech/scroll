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
	bytes, err := chunk.Encode(0)
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
	bytes, err = chunk.Encode(0)
	assert.Nil(t, bytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number of blocks exceeds 1 byte")

	// Test case 3: when the chunk contains one block.
	templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_02.json")
	assert.NoError(t, err)

	wrappedBlock := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock))
	assert.Equal(t, uint64(0), wrappedBlock.NumL1Messages(0))
	assert.Equal(t, uint64(358), wrappedBlock.EstimateL1CommitCalldataSize())
	assert.Equal(t, uint64(2), wrappedBlock.NumL2Transactions())
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock,
		},
	}
	assert.Equal(t, uint64(0), chunk.NumL1Messages(0))
	assert.Equal(t, uint64(6966), chunk.EstimateL1CommitGas())
	bytes, err = chunk.Encode(0)
	hexString := hex.EncodeToString(bytes)
	assert.NoError(t, err)
	assert.Equal(t, 299, len(bytes))
	assert.Equal(t, "0100000000000000020000000063807b2a0000000000000000000000000000000000000000000000000000000000000000000355418d1e81840002000000000073f87180843b9aec2e8307a12094c0c4c8baea3f6acb49b6e1fb9e2adeceeacb0ca28a152d02c7e14af60000008083019ecea0ab07ae99c67aa78e7ba5cf6781e90cc32b219b1de102513d56548a41e86df514a034cbd19feacd73e8ce64d00c4d1996b9b5243c578fd7f51bfaec288bbaf42a8b00000073f87101843b9aec2e8307a1209401bae6bf68e9a03fb2bc0615b1bf0d69ce9411ed8a152d02c7e14af60000008083019ecea0f039985866d8256f10c1be4f7b2cace28d8f20bde27e2604393eb095b7f77316a05a3e6e81065f2b4604bcec5bd4aba684835996fc3f879380aac1c09c6eed32f1", hexString)

	// Test case 4: when the chunk contains one block with 1 L1MsgTx
	templateBlockTrace2, err := os.ReadFile("../testdata/blockTrace_04.json")
	assert.NoError(t, err)

	wrappedBlock2 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace2, wrappedBlock2))
	assert.Equal(t, uint64(11), wrappedBlock2.NumL1Messages(0)) // 0..=9 skipped, 10 included
	assert.Equal(t, uint64(96), wrappedBlock2.EstimateL1CommitCalldataSize())
	assert.Equal(t, uint64(1), wrappedBlock2.NumL2Transactions())
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock2,
		},
	}
	assert.Equal(t, uint64(11), chunk.NumL1Messages(0))
	assert.Equal(t, uint64(5002), chunk.EstimateL1CommitGas())
	bytes, err = chunk.Encode(0)
	hexString = hex.EncodeToString(bytes)
	assert.NoError(t, err)
	assert.Equal(t, 97, len(bytes))
	assert.Equal(t, "01000000000000000d00000000646b6e13000000000000000000000000000000000000000000000000000000000000000000000000007a1200000c000b00000020df0b80825dc0941a258d17bf244c4df02d40343a7626a9d321e1058080808080", hexString)

	// Test case 5: when the chunk contains two blocks each with 1 L1MsgTx
	// TODO: revise this test, we cannot reuse the same L1MsgTx twice
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock2,
			wrappedBlock2,
		},
	}
	assert.Equal(t, uint64(11), chunk.NumL1Messages(0))
	assert.Equal(t, uint64(9958), chunk.EstimateL1CommitGas())
	bytes, err = chunk.Encode(0)
	hexString = hex.EncodeToString(bytes)
	assert.NoError(t, err)
	assert.Equal(t, 193, len(bytes))
	assert.Equal(t, "02000000000000000d00000000646b6e13000000000000000000000000000000000000000000000000000000000000000000000000007a1200000c000b000000000000000d00000000646b6e13000000000000000000000000000000000000000000000000000000000000000000000000007a12000001000000000020df0b80825dc0941a258d17bf244c4df02d40343a7626a9d321e105808080808000000020df0b80825dc0941a258d17bf244c4df02d40343a7626a9d321e1058080808080", hexString)
}

func TestChunkHash(t *testing.T) {
	// Test case 1: when the chunk contains no blocks
	chunk := &Chunk{
		Blocks: []*WrappedBlock{},
	}
	hash, err := chunk.Hash(0)
	assert.Error(t, err)
	assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", hash.Hex())
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
	hash, err = chunk.Hash(0)
	assert.NoError(t, err)
	assert.Equal(t, "0x78c839dfc494396c16b40946f32b3f4c3e8c2d4bfd04aefcf235edec474482f8", hash.Hex())

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
	hash, err = chunk.Hash(0)
	assert.NoError(t, err)
	assert.Equal(t, "0xaa9e494f72bc6965857856f0fae6916f27b2a6591c714a573b2fab46df03b8ae", hash.Hex())

	// Test case 4: successfully hashing a chunk on two blocks each with L1 and L2 txs
	templateBlockTrace2, err := os.ReadFile("../testdata/blockTrace_04.json")
	assert.NoError(t, err)
	wrappedBlock2 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace2, wrappedBlock2))
	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock2,
			wrappedBlock2,
		},
	}
	hash, err = chunk.Hash(0)
	assert.NoError(t, err)
	assert.Equal(t, "0x2eb7dd63bf8fc29a0f8c10d16c2ae6f9da446907c79d50f5c164d30dc8526b60", hash.Hex())
}

func TestErrorPaths(t *testing.T) {
	// test 1: Header.Number is not a uint64
	templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_02.json")
	assert.NoError(t, err)

	wrappedBlock := &WrappedBlock{}

	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock))
	wrappedBlock.Header.Number = wrappedBlock.Header.Number.Lsh(wrappedBlock.Header.Number, 64)
	bytes, err := wrappedBlock.Encode(0)
	assert.Nil(t, bytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "block number is not uint64")

	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock))
	for i := 0; i < 65537; i++ {
		wrappedBlock.Transactions = append(wrappedBlock.Transactions, wrappedBlock.Transactions[0])
	}

	bytes, err = wrappedBlock.Encode(0)
	assert.Nil(t, bytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number of transactions exceeds max uint16")

	chunk := &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock,
		},
	}

	bytes, err = chunk.Encode(0)
	assert.Nil(t, bytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number of transactions exceeds max uint16")

	wrappedBlock.Transactions = wrappedBlock.Transactions[:1]
	wrappedBlock.Transactions[0].Data = "not-a-hex"
	bytes, err = chunk.Encode(0)
	assert.Nil(t, bytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hex string without 0x prefix")

	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock))
	wrappedBlock.Transactions[0].TxHash = "not-a-hex"
	_, err = chunk.Hash(0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid byte")

	templateBlockTrace2, err := os.ReadFile("../testdata/blockTrace_04.json")
	assert.NoError(t, err)

	wrappedBlock2 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace2, wrappedBlock2))
	for i := 0; i < 65535; i++ {
		tx := &wrappedBlock2.Transactions[i]
		txCopy := *tx
		txCopy.Nonce = uint64(i + 1)
		wrappedBlock2.Transactions = append(wrappedBlock2.Transactions, txCopy)
	}

	bytes, err = wrappedBlock2.Encode(0)
	assert.Nil(t, bytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number of L1 messages exceeds max uint16")

	chunk = &Chunk{
		Blocks: []*WrappedBlock{
			wrappedBlock2,
		},
	}

	bytes, err = chunk.Encode(0)
	assert.Nil(t, bytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number of L1 messages exceeds max uint16")

}
