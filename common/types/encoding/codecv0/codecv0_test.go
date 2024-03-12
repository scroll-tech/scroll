package codecv0

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types/encoding"
)

func TestCodecV0(t *testing.T) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	// Test case: when the batch and chunk contains one block.
	templateBlockTrace, err := os.ReadFile("../../../testdata/blockTrace_02.json")
	assert.NoError(t, err)

	block1 := &encoding.Block{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, block1))
	assert.Equal(t, uint64(298), EstimateBlockL1CommitCalldataSize(block1))
	assert.Equal(t, uint64(4900), EstimateBlockL1CommitGas(block1))

	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{block1},
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
		TotalL1MessagePoppedBefore: 1,
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
	assert.Equal(t, "000000000000000000000000000000000000000000000000018fbc5eecfefc5bd9d1618ecef1fed160a7838448383595a2257d4c9bd5c5fa3e0000000000000000000000000000000000000000000000000000000000000000", batchHexString)
	assert.Equal(t, 0, len(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(1), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(0), daBatch.L1MessagePopped)

	decodedDABatch := MustNewDABatchFromBytes(batchBytes)
	decodedBatchBytes := decodedDABatch.Encode()
	decodedBatchHexString := hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)

	// Test case: when the batch and chunk contains two block.
	templateBlockTrace, err = os.ReadFile("../../../testdata/blockTrace_03.json")
	assert.NoError(t, err)

	block2 := &encoding.Block{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, block2))
	assert.Equal(t, uint64(5737), EstimateBlockL1CommitCalldataSize(block2))
	assert.Equal(t, uint64(93485), EstimateBlockL1CommitGas(block2))

	chunk = &encoding.Chunk{
		Blocks: []*encoding.Block{block1, block2},
	}
	assert.Equal(t, uint64(6035), EstimateChunkL1CommitCalldataSize(chunk))
	assert.Equal(t, uint64(100614), EstimateChunkL1CommitGas(chunk))

	daChunk = NewDAChunk(chunk, 0)
	chunkBytes = daChunk.Encode()
	assert.Equal(t, 6036, len(chunkBytes))

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunk.Hash(),
		EndChunkHash:               daChunk.Hash(),
	}

	assert.Equal(t, uint64(6035), EstimateBatchL1CommitCalldataSize(batch))
	assert.Equal(t, uint64(257769), EstimateBatchL1CommitGas(batch))

	daBatch = NewDABatch(batch)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 89, len(batchBytes))
	assert.Equal(t, "0000000000000000000000000000000000000000000000000057a3f6cb52ad8d9ae9775a2780a528ef3b5715abe375724e8fc5d2a15d101f7d0000000000000000000000000000000000000000000000000000000000000000", batchHexString)
	assert.Equal(t, 0, len(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(0), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(0), daBatch.L1MessagePopped)

	decodedDABatch = MustNewDABatchFromBytes(batchBytes)
	decodedBatchBytes = decodedDABatch.Encode()
	decodedBatchHexString = hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)

	// Test case: when the chunk contains one block with 1 L1MsgTx.
	templateBlockTrace, err = os.ReadFile("../../../testdata/blockTrace_04.json")
	assert.NoError(t, err)

	block3 := &encoding.Block{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, block3))
	assert.Equal(t, uint64(96), EstimateBlockL1CommitCalldataSize(block3))
	assert.Equal(t, uint64(4187), EstimateBlockL1CommitGas(block3))

	chunk = &encoding.Chunk{
		Blocks: []*encoding.Block{block3},
	}
	assert.Equal(t, uint64(96), EstimateChunkL1CommitCalldataSize(chunk))
	assert.Equal(t, uint64(5329), EstimateChunkL1CommitGas(chunk))

	daChunk = NewDAChunk(chunk, 0)
	chunkBytes = daChunk.Encode()
	chunkHexString = hex.EncodeToString(chunkBytes)
	assert.Equal(t, 97, len(chunkBytes))
	assert.Equal(t, "01000000000000000d00000000646b6e13000000000000000000000000000000000000000000000000000000000000000000000000007a1200000c000b00000020df0b80825dc0941a258d17bf244c4df02d40343a7626a9d321e1058080808080", chunkHexString)

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunk.Hash(),
		EndChunkHash:               daChunk.Hash(),
	}

	assert.Equal(t, uint64(96), EstimateBatchL1CommitCalldataSize(batch))
	assert.Equal(t, uint64(161889), EstimateBatchL1CommitGas(batch))

	daBatch = NewDABatch(batch)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 121, len(batchBytes))
	assert.Equal(t, "000000000000000000000000000000000b000000000000000b34f419ce7e882295bdb5aec6cce56ffa788a5fed4744d7fbd77e4acbf409f1ca000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003ff", batchHexString)
	assert.Equal(t, 32, len(daBatch.SkippedL1MessageBitmap))
	expectedBitmap := "00000000000000000000000000000000000000000000000000000000000003ff"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(11), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(11), daBatch.L1MessagePopped)

	decodedDABatch = MustNewDABatchFromBytes(batchBytes)
	decodedBatchBytes = decodedDABatch.Encode()
	decodedBatchHexString = hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)

	// Test case: batch contains multiple chunks, chunk contains multiple blocks.
	templateBlockTrace, err = os.ReadFile("../../../testdata/blockTrace_05.json")
	assert.NoError(t, err)

	block4 := &encoding.Block{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, block4))
	assert.Equal(t, uint64(60), EstimateBlockL1CommitCalldataSize(block4))
	assert.Equal(t, uint64(14020), EstimateBlockL1CommitGas(block4))

	chunk1 := &encoding.Chunk{
		Blocks: []*encoding.Block{block1, block2, block3},
	}
	assert.Equal(t, uint64(6131), EstimateChunkL1CommitCalldataSize(chunk1))
	assert.Equal(t, uint64(105897), EstimateChunkL1CommitGas(chunk1))

	daChunk1 := NewDAChunk(chunk1, 0)
	chunkBytes1 := daChunk1.Encode()
	assert.Equal(t, 6132, len(chunkBytes1))

	chunk2 := &encoding.Chunk{
		Blocks: []*encoding.Block{block4},
	}
	assert.Equal(t, uint64(60), EstimateChunkL1CommitCalldataSize(chunk2))
	assert.Equal(t, uint64(15189), EstimateChunkL1CommitGas(chunk2))

	daChunk2 := NewDAChunk(chunk2, 0)
	chunkBytes2 := daChunk2.Encode()
	assert.Equal(t, 61, len(chunkBytes2))

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk1, chunk2},
		StartChunkIndex:            0,
		EndChunkIndex:              1,
		StartChunkHash:             daChunk1.Hash(),
		EndChunkHash:               daChunk2.Hash(),
	}

	assert.Equal(t, uint64(6191), EstimateBatchL1CommitCalldataSize(batch))
	assert.Equal(t, uint64(278926), EstimateBatchL1CommitGas(batch))

	daBatch = NewDABatch(batch)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 121, len(batchBytes))
	assert.Equal(t, "000000000000000000000000000000002a000000000000002a6ef79114e7d29ab5af21a6553ed3693aa5e524be5a8506beb4f13cf3236edaba00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001ffffffbff", batchHexString)
	assert.Equal(t, 32, len(daBatch.SkippedL1MessageBitmap))
	expectedBitmap = "0000000000000000000000000000000000000000000000000000001ffffffbff"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(42), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(42), daBatch.L1MessagePopped)

	// Test case: many consecutive L1 Msgs in 1 bitmap, no leading skipped msgs.
	chunk = &encoding.Chunk{
		Blocks: []*encoding.Block{block4},
	}
	assert.Equal(t, uint64(60), EstimateChunkL1CommitCalldataSize(chunk))
	assert.Equal(t, uint64(15189), EstimateChunkL1CommitGas(chunk))

	daChunk = NewDAChunk(chunk, 0)
	chunkBytes = daChunk.Encode()
	assert.Equal(t, 61, len(chunkBytes))

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 37,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunk.Hash(),
		EndChunkHash:               daChunk.Hash(),
	}

	assert.Equal(t, uint64(60), EstimateBatchL1CommitCalldataSize(batch))
	assert.Equal(t, uint64(171730), EstimateBatchL1CommitGas(batch))

	daBatch = NewDABatch(batch)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 121, len(batchBytes))
	assert.Equal(t, "0000000000000000000000000000000005000000000000002ac62fb58ec2d5393e00960f1cc23cab883b685296efa03d13ea2dd4c6de79cc5500000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", batchHexString)
	assert.Equal(t, 32, len(daBatch.SkippedL1MessageBitmap))
	expectedBitmap = "0000000000000000000000000000000000000000000000000000000000000000"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(42), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(5), daBatch.L1MessagePopped)

	// Test case: many sparse L1 Msgs in 1 bitmap.
	templateBlockTrace, err = os.ReadFile("../../../testdata/blockTrace_06.json")
	assert.NoError(t, err)

	block5 := &encoding.Block{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, block5))
	assert.Equal(t, uint64(60), EstimateBlockL1CommitCalldataSize(block5))
	assert.Equal(t, uint64(8796), EstimateBlockL1CommitGas(block5))

	chunk = &encoding.Chunk{
		Blocks: []*encoding.Block{block5},
	}
	assert.Equal(t, uint64(60), EstimateChunkL1CommitCalldataSize(chunk))
	assert.Equal(t, uint64(9947), EstimateChunkL1CommitGas(chunk))

	daChunk = NewDAChunk(chunk, 0)
	chunkBytes = daChunk.Encode()
	assert.Equal(t, 61, len(chunkBytes))

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunk.Hash(),
		EndChunkHash:               daChunk.Hash(),
	}

	assert.Equal(t, uint64(60), EstimateBatchL1CommitCalldataSize(batch))
	assert.Equal(t, uint64(166504), EstimateBatchL1CommitGas(batch))

	daBatch = NewDABatch(batch)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 121, len(batchBytes))
	assert.Equal(t, "000000000000000000000000000000000a000000000000000ac7bcc8da943dd83404e84d9ce7e894ab97ce4829df4eb51ebbbe13c90b5a3f4d000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001dd", batchHexString)
	assert.Equal(t, 32, len(daBatch.SkippedL1MessageBitmap))
	expectedBitmap = "00000000000000000000000000000000000000000000000000000000000001dd"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(10), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(10), daBatch.L1MessagePopped)

	// Test case: many L1 Msgs in each of 2 bitmaps.
	templateBlockTrace, err = os.ReadFile("../../../testdata/blockTrace_07.json")
	assert.NoError(t, err)

	block6 := &encoding.Block{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, block6))
	assert.Equal(t, uint64(60), EstimateBlockL1CommitCalldataSize(block6))
	assert.Equal(t, uint64(6184), EstimateBlockL1CommitGas(block6))

	chunk = &encoding.Chunk{
		Blocks: []*encoding.Block{block6},
	}
	assert.Equal(t, uint64(60), EstimateChunkL1CommitCalldataSize(chunk))
	assert.Equal(t, uint64(7326), EstimateChunkL1CommitGas(chunk))

	daChunk = NewDAChunk(chunk, 0)
	chunkBytes = daChunk.Encode()
	assert.Equal(t, 61, len(chunkBytes))

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunk.Hash(),
		EndChunkHash:               daChunk.Hash(),
	}

	assert.Equal(t, uint64(60), EstimateBatchL1CommitCalldataSize(batch))
	assert.Equal(t, uint64(164388), EstimateBatchL1CommitGas(batch))

	daBatch = NewDABatch(batch)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 153, len(batchBytes))
	assert.Equal(t, "00000000000000000000000000000001010000000000000101899a411a3309c6491701b7b955c7b1115ac015414bbb71b59a0ca561668d52080000000000000000000000000000000000000000000000000000000000000000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd0000000000000000000000000000000000000000000000000000000000000000", batchHexString)
	assert.Equal(t, 64, len(daBatch.SkippedL1MessageBitmap))
	expectedBitmap = "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd0000000000000000000000000000000000000000000000000000000000000000"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(257), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(257), daBatch.L1MessagePopped)
}
