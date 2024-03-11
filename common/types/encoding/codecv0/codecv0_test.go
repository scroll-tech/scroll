package codecv0

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types/encoding"
)

func TestCodecV0(t *testing.T) {
	// Test case: when the batch and chunk contains one block.
	templateBlockTrace, err := os.ReadFile("../../../testdata/blockTrace_02.json")
	assert.NoError(t, err)

	block := &encoding.Block{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, block))
	assert.Equal(t, uint64(298), EstimateBlockL1CommitCalldataSize(block))
	assert.Equal(t, uint64(4900), EstimateBlockL1CommitGas(block))

	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{block},
	}
	assert.Equal(t, uint64(298), EstimateChunkL1CommitCalldataSize(chunk))
	assert.Equal(t, uint64(6042), EstimateChunkL1CommitGas(chunk))

	daChunk := NewDAChunk(chunk, 0)
	chunkBytes := daChunk.Encode()
	chunkHexString := hex.EncodeToString(chunkBytes)
	assert.Equal(t, 299, len(chunkBytes))
	assert.Equal(t, "0100000000000000020000000063807b2a0000000000000000000000000000000000000000000000000000000000001de9000355418d1e81840002000000000073f87180843b9aec2e8307a12094c0c4c8baea3f6acb49b6e1fb9e2adeceeacb0ca28a152d02c7e14af60000008083019ecea0ab07ae99c67aa78e7ba5cf6781e90cc32b219b1de102513d56548a41e86df514a034cbd19feacd73e8ce64d00c4d1996b9b5243c578fd7f51bfaec288bbaf42a8b00000073f87101843b9aec2e8307a1209401bae6bf68e9a03fb2bc0615b1bf0d69ce9411ed8a152d02c7e14af60000008083019ecea0f039985866d8256f10c1be4f7b2cace28d8f20bde27e2604393eb095b7f77316a05a3e6e81065f2b4604bcec5bd4aba684835996fc3f879380aac1c09c6eed32f1", chunkHexString)

	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunk.Hash(),
		EndChunkHash:               daChunk.Hash(),
	}

	assert.Equal(t, uint64(298), EstimateBatchL1CommitCalldataSize(batch))
	assert.Equal(t, uint64(162591), EstimateBatchL1CommitGas(batch))

	daBatch := NewDABatch(batch)
	batchBytes := daBatch.Encode()
	batchHexString := hex.EncodeToString(batchBytes)
	assert.Equal(t, 89, len(batchBytes))
	assert.Equal(t, "000000000000000000000000000000000000000000000000008fbc5eecfefc5bd9d1618ecef1fed160a7838448383595a2257d4c9bd5c5fa3e0000000000000000000000000000000000000000000000000000000000000000", batchHexString)

	decodedDABatch := MustNewDABatchFromBytes(batchBytes)
	decodedBatchBytes := decodedDABatch.Encode()
	decodedBatchHexString := hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)
}
