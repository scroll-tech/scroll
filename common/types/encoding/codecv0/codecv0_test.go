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

	block1 := readBlockFromJSON(t, "../../../testdata/blockTrace_02.json")
	block2 := readBlockFromJSON(t, "../../../testdata/blockTrace_03.json")
	block3 := readBlockFromJSON(t, "../../../testdata/blockTrace_04.json")
	block4 := readBlockFromJSON(t, "../../../testdata/blockTrace_05.json")
	block5 := readBlockFromJSON(t, "../../../testdata/blockTrace_06.json")
	block6 := readBlockFromJSON(t, "../../../testdata/blockTrace_07.json")

	blockL1CommitCalldataSize, err := EstimateBlockL1CommitCalldataSize(block1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(298), blockL1CommitCalldataSize)
	blockL1CommitGas, err := EstimateBlockL1CommitGas(block1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(4900), blockL1CommitGas)
	blockL1CommitCalldataSize, err = EstimateBlockL1CommitCalldataSize(block2)
	assert.NoError(t, err)
	assert.Equal(t, uint64(5745), blockL1CommitCalldataSize)
	blockL1CommitGas, err = EstimateBlockL1CommitGas(block2)
	assert.NoError(t, err)
	assert.Equal(t, uint64(93613), blockL1CommitGas)
	blockL1CommitCalldataSize, err = EstimateBlockL1CommitCalldataSize(block3)
	assert.NoError(t, err)
	assert.Equal(t, uint64(96), blockL1CommitCalldataSize)
	blockL1CommitGas, err = EstimateBlockL1CommitGas(block3)
	assert.NoError(t, err)
	assert.Equal(t, uint64(4187), blockL1CommitGas)
	blockL1CommitCalldataSize, err = EstimateBlockL1CommitCalldataSize(block4)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), blockL1CommitCalldataSize)
	blockL1CommitGas, err = EstimateBlockL1CommitGas(block4)
	assert.NoError(t, err)
	assert.Equal(t, uint64(14020), blockL1CommitGas)
	blockL1CommitCalldataSize, err = EstimateBlockL1CommitCalldataSize(block5)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), blockL1CommitCalldataSize)
	blockL1CommitGas, err = EstimateBlockL1CommitGas(block5)
	assert.NoError(t, err)
	assert.Equal(t, uint64(8796), blockL1CommitGas)
	blockL1CommitCalldataSize, err = EstimateBlockL1CommitCalldataSize(block6)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), blockL1CommitCalldataSize)
	blockL1CommitGas, err = EstimateBlockL1CommitGas(block6)
	assert.NoError(t, err)
	assert.Equal(t, uint64(6184), blockL1CommitGas)

	// Test case: when the batch and chunk contains one block.
	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{block1},
	}
	chunkL1CommitCalldataSize, err := EstimateChunkL1CommitCalldataSize(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(298), chunkL1CommitCalldataSize)
	chunkL1CommitGas, err := EstimateChunkL1CommitGas(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(6042), chunkL1CommitGas)

	daChunk, err := NewDAChunk(chunk, 0)
	assert.NoError(t, err)
	chunkBytes, err := daChunk.Encode()
	assert.NoError(t, err)
	chunkHexString := hex.EncodeToString(chunkBytes)
	assert.Equal(t, 299, len(chunkBytes))
	assert.Equal(t, "0100000000000000020000000063807b2a0000000000000000000000000000000000000000000000000000000000001de9000355418d1e81840002000000000073f87180843b9aec2e8307a12094c0c4c8baea3f6acb49b6e1fb9e2adeceeacb0ca28a152d02c7e14af60000008083019ecea0ab07ae99c67aa78e7ba5cf6781e90cc32b219b1de102513d56548a41e86df514a034cbd19feacd73e8ce64d00c4d1996b9b5243c578fd7f51bfaec288bbaf42a8b00000073f87101843b9aec2e8307a1209401bae6bf68e9a03fb2bc0615b1bf0d69ce9411ed8a152d02c7e14af60000008083019ecea0f039985866d8256f10c1be4f7b2cace28d8f20bde27e2604393eb095b7f77316a05a3e6e81065f2b4604bcec5bd4aba684835996fc3f879380aac1c09c6eed32f1", chunkHexString)
	daChunkHash, err := daChunk.Hash()
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0xde642c68122634b33fa1e6e4243b17be3bfd0dc6f996f204ef6d7522516bd840"), daChunkHash)

	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 1,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunkHash,
		EndChunkHash:               daChunkHash,
	}

	batchL1CommitCalldataSize, err := EstimateBatchL1CommitCalldataSize(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(298), batchL1CommitCalldataSize)
	batchL1CommitGas, err := EstimateBatchL1CommitGas(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(162591), batchL1CommitGas)

	daBatch, err := NewDABatch(batch)
	assert.NoError(t, err)
	batchBytes := daBatch.Encode()
	batchHexString := hex.EncodeToString(batchBytes)
	assert.Equal(t, 89, len(batchBytes))
	assert.Equal(t, "000000000000000000000000000000000000000000000000018fbc5eecfefc5bd9d1618ecef1fed160a7838448383595a2257d4c9bd5c5fa3e0000000000000000000000000000000000000000000000000000000000000000", batchHexString)
	assert.Equal(t, 0, len(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(1), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(0), daBatch.L1MessagePopped)
	assert.Equal(t, common.HexToHash("0x5c799a5938f7885c9476b5868c95b0d23caa7caf3b7d61dfd3449ca222fe2ea2"), daBatch.Hash())

	decodedDABatch, err := NewDABatchFromBytes(batchBytes)
	assert.NoError(t, err)
	decodedBatchBytes := decodedDABatch.Encode()
	decodedBatchHexString := hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)

	// Test case: when the batch and chunk contains two block.
	chunk = &encoding.Chunk{
		Blocks: []*encoding.Block{block1, block2},
	}
	chunkL1CommitCalldataSize, err = EstimateChunkL1CommitCalldataSize(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(6043), chunkL1CommitCalldataSize)
	chunkL1CommitGas, err = EstimateChunkL1CommitGas(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100742), chunkL1CommitGas)

	daChunk, err = NewDAChunk(chunk, 0)
	assert.NoError(t, err)
	chunkBytes, err = daChunk.Encode()
	assert.NoError(t, err)
	assert.Equal(t, 6044, len(chunkBytes))
	daChunkHash, err = daChunk.Hash()
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0x014916a83eccdb0d01e814b4d4ab90eb9049ba9a3cb0994919b86ad873bcd028"), daChunkHash)

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunkHash,
		EndChunkHash:               daChunkHash,
	}

	batchL1CommitCalldataSize, err = EstimateBatchL1CommitCalldataSize(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(6043), batchL1CommitCalldataSize)
	batchL1CommitGas, err = EstimateBatchL1CommitGas(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(257897), batchL1CommitGas)

	daBatch, err = NewDABatch(batch)
	assert.NoError(t, err)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 89, len(batchBytes))
	assert.Equal(t, "0000000000000000000000000000000000000000000000000074dd561a36921590926bee01fd0d53747c5f3e48e48a2d5538b9ab0e1511cfd70000000000000000000000000000000000000000000000000000000000000000", batchHexString)
	assert.Equal(t, 0, len(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(0), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(0), daBatch.L1MessagePopped)
	assert.Equal(t, common.HexToHash("0x926ffa923e6b5ea7311351cf6401b1ee3ef736faf7afd8e7d7f712cfd021a192"), daBatch.Hash())

	decodedDABatch, err = NewDABatchFromBytes(batchBytes)
	assert.NoError(t, err)
	decodedBatchBytes = decodedDABatch.Encode()
	decodedBatchHexString = hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)

	// Test case: when the chunk contains one block with 1 L1MsgTx.
	chunk = &encoding.Chunk{
		Blocks: []*encoding.Block{block3},
	}
	chunkL1CommitCalldataSize, err = EstimateChunkL1CommitCalldataSize(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(96), chunkL1CommitCalldataSize)
	chunkL1CommitGas, err = EstimateChunkL1CommitGas(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(5329), chunkL1CommitGas)

	daChunk, err = NewDAChunk(chunk, 0)
	assert.NoError(t, err)
	chunkBytes, err = daChunk.Encode()
	assert.NoError(t, err)
	chunkHexString = hex.EncodeToString(chunkBytes)
	assert.Equal(t, 97, len(chunkBytes))
	assert.Equal(t, "01000000000000000d00000000646b6e13000000000000000000000000000000000000000000000000000000000000000000000000007a1200000c000b00000020df0b80825dc0941a258d17bf244c4df02d40343a7626a9d321e1058080808080", chunkHexString)
	daChunkHash, err = daChunk.Hash()
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0x9e643c8a9203df542e39d9bfdcb07c99575b3c3d557791329fef9d83cc4147d0"), daChunkHash)

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunkHash,
		EndChunkHash:               daChunkHash,
	}

	batchL1CommitCalldataSize, err = EstimateBatchL1CommitCalldataSize(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(96), batchL1CommitCalldataSize)
	batchL1CommitGas, err = EstimateBatchL1CommitGas(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(161889), batchL1CommitGas)

	daBatch, err = NewDABatch(batch)
	assert.NoError(t, err)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 121, len(batchBytes))
	assert.Equal(t, "000000000000000000000000000000000b000000000000000b34f419ce7e882295bdb5aec6cce56ffa788a5fed4744d7fbd77e4acbf409f1ca000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003ff", batchHexString)
	assert.Equal(t, 32, len(daBatch.SkippedL1MessageBitmap))
	expectedBitmap := "00000000000000000000000000000000000000000000000000000000000003ff"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(11), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(11), daBatch.L1MessagePopped)
	assert.Equal(t, common.HexToHash("0xfbb081f25d6d06aefd76f062eee50885faf5bb050c8f31d533fc8560e655b690"), daBatch.Hash())

	decodedDABatch, err = NewDABatchFromBytes(batchBytes)
	assert.NoError(t, err)
	decodedBatchBytes = decodedDABatch.Encode()
	decodedBatchHexString = hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)

	// Test case: batch contains multiple chunks, chunk contains multiple blocks.
	chunk1 := &encoding.Chunk{
		Blocks: []*encoding.Block{block1, block2, block3},
	}
	chunk1L1CommitCalldataSize, err := EstimateChunkL1CommitCalldataSize(chunk1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(6139), chunk1L1CommitCalldataSize)
	chunk1L1CommitGas, err := EstimateChunkL1CommitGas(chunk1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(106025), chunk1L1CommitGas)

	daChunk1, err := NewDAChunk(chunk1, 0)
	assert.NoError(t, err)
	chunkBytes1, err := daChunk1.Encode()
	assert.NoError(t, err)
	assert.Equal(t, 6140, len(chunkBytes1))

	chunk2 := &encoding.Chunk{
		Blocks: []*encoding.Block{block4},
	}
	chunk2L1CommitCalldataSize, err := EstimateChunkL1CommitCalldataSize(chunk2)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), chunk2L1CommitCalldataSize)
	chunk2L1CommitGas, err := EstimateChunkL1CommitGas(chunk2)
	assert.NoError(t, err)
	assert.Equal(t, uint64(15189), chunk2L1CommitGas)

	daChunk2, err := NewDAChunk(chunk2, 0)
	assert.NoError(t, err)
	chunkBytes2, err := daChunk2.Encode()
	assert.NoError(t, err)
	assert.Equal(t, 61, len(chunkBytes2))

	daChunk1Hash, err := daChunk1.Hash()
	assert.NoError(t, err)
	daChunk2Hash, err := daChunk2.Hash()
	assert.NoError(t, err)

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk1, chunk2},
		StartChunkIndex:            0,
		EndChunkIndex:              1,
		StartChunkHash:             daChunk1Hash,
		EndChunkHash:               daChunk2Hash,
	}

	batchL1CommitCalldataSize, err = EstimateBatchL1CommitCalldataSize(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(6199), batchL1CommitCalldataSize)
	batchL1CommitGas, err = EstimateBatchL1CommitGas(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(279054), batchL1CommitGas)

	daBatch, err = NewDABatch(batch)
	assert.NoError(t, err)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 121, len(batchBytes))
	assert.Equal(t, "000000000000000000000000000000002a000000000000002a1f9b3d942a6ee14e7afc52225c91fa44faa0a7ec511df9a2d9348d33bcd142fc00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001ffffffbff", batchHexString)
	assert.Equal(t, 32, len(daBatch.SkippedL1MessageBitmap))
	expectedBitmap = "0000000000000000000000000000000000000000000000000000001ffffffbff"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(42), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(42), daBatch.L1MessagePopped)
	assert.Equal(t, common.HexToHash("0xc5e787fa6a83374135c3b95bd8325bcc0440cd5eb2d71bb31ddca67dd2d44f64"), daBatch.Hash())

	decodedDABatch, err = NewDABatchFromBytes(batchBytes)
	assert.NoError(t, err)
	decodedBatchBytes = decodedDABatch.Encode()
	decodedBatchHexString = hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)

	// Test case: many consecutive L1 Msgs in 1 bitmap, no leading skipped msgs.
	chunk = &encoding.Chunk{
		Blocks: []*encoding.Block{block4},
	}
	chunkL1CommitCalldataSize, err = EstimateChunkL1CommitCalldataSize(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), chunkL1CommitCalldataSize)
	chunkL1CommitGas, err = EstimateChunkL1CommitGas(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(15189), chunkL1CommitGas)

	daChunk, err = NewDAChunk(chunk, 0)
	assert.NoError(t, err)
	chunkBytes, err = daChunk.Encode()
	assert.NoError(t, err)
	assert.Equal(t, 61, len(chunkBytes))
	daChunkHash, err = daChunk.Hash()
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0x854fc3136f47ce482ec85ee3325adfa16a1a1d60126e1c119eaaf0c3a9e90f8e"), daChunkHash)

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 37,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunkHash,
		EndChunkHash:               daChunkHash,
	}

	batchL1CommitCalldataSize, err = EstimateBatchL1CommitCalldataSize(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), batchL1CommitCalldataSize)
	batchL1CommitGas, err = EstimateBatchL1CommitGas(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(171730), batchL1CommitGas)

	daBatch, err = NewDABatch(batch)
	assert.NoError(t, err)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 121, len(batchBytes))
	assert.Equal(t, "0000000000000000000000000000000005000000000000002ac62fb58ec2d5393e00960f1cc23cab883b685296efa03d13ea2dd4c6de79cc5500000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", batchHexString)
	assert.Equal(t, 32, len(daBatch.SkippedL1MessageBitmap))
	expectedBitmap = "0000000000000000000000000000000000000000000000000000000000000000"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(42), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(5), daBatch.L1MessagePopped)
	assert.Equal(t, common.HexToHash("0x1b62133deff60768285538373754403ac4c792c371ff38c24151e8c0bcaecb41"), daBatch.Hash())

	decodedDABatch, err = NewDABatchFromBytes(batchBytes)
	assert.NoError(t, err)
	decodedBatchBytes = decodedDABatch.Encode()
	decodedBatchHexString = hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)

	// Test case: many consecutive L1 Msgs in 1 bitmap, with leading skipped msgs.
	chunk = &encoding.Chunk{
		Blocks: []*encoding.Block{block4},
	}
	chunkL1CommitCalldataSize, err = EstimateChunkL1CommitCalldataSize(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), chunkL1CommitCalldataSize)
	chunkL1CommitGas, err = EstimateChunkL1CommitGas(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(15189), chunkL1CommitGas)

	daChunk, err = NewDAChunk(chunk, 0)
	assert.NoError(t, err)
	chunkBytes, err = daChunk.Encode()
	assert.NoError(t, err)
	assert.Equal(t, 61, len(chunkBytes))
	daChunkHash, err = daChunk.Hash()
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0x854fc3136f47ce482ec85ee3325adfa16a1a1d60126e1c119eaaf0c3a9e90f8e"), daChunkHash)

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunkHash,
		EndChunkHash:               daChunkHash,
	}

	batchL1CommitCalldataSize, err = EstimateBatchL1CommitCalldataSize(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), batchL1CommitCalldataSize)
	batchL1CommitGas, err = EstimateBatchL1CommitGas(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(171810), batchL1CommitGas)

	daBatch, err = NewDABatch(batch)
	assert.NoError(t, err)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 121, len(batchBytes))
	assert.Equal(t, "000000000000000000000000000000002a000000000000002a93255aa24dd468c5645f1e6901b8131a7a78a0eeb2a17cbb09ba64688a8de6b400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001fffffffff", batchHexString)
	assert.Equal(t, 32, len(daBatch.SkippedL1MessageBitmap))
	expectedBitmap = "0000000000000000000000000000000000000000000000000000001fffffffff"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(42), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(42), daBatch.L1MessagePopped)
	assert.Equal(t, common.HexToHash("0x99f9648e4d090f1222280bec95a3f1e39c6cbcd4bff21eb2ae94b1536bb23acc"), daBatch.Hash())

	decodedDABatch, err = NewDABatchFromBytes(batchBytes)
	assert.NoError(t, err)
	decodedBatchBytes = decodedDABatch.Encode()
	decodedBatchHexString = hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)

	// Test case: many sparse L1 Msgs in 1 bitmap.
	chunk = &encoding.Chunk{
		Blocks: []*encoding.Block{block5},
	}
	chunkL1CommitCalldataSize, err = EstimateChunkL1CommitCalldataSize(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), chunkL1CommitCalldataSize)
	chunkL1CommitGas, err = EstimateChunkL1CommitGas(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(9947), chunkL1CommitGas)

	daChunk, err = NewDAChunk(chunk, 0)
	assert.NoError(t, err)
	chunkBytes, err = daChunk.Encode()
	assert.NoError(t, err)
	assert.Equal(t, 61, len(chunkBytes))
	daChunkHash, err = daChunk.Hash()
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0x2aa220ca7bd1368e59e8053eb3831e30854aa2ec8bd3af65cee350c1c0718ba6"), daChunkHash)

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunkHash,
		EndChunkHash:               daChunkHash,
	}

	batchL1CommitCalldataSize, err = EstimateBatchL1CommitCalldataSize(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), batchL1CommitCalldataSize)
	batchL1CommitGas, err = EstimateBatchL1CommitGas(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(166504), batchL1CommitGas)

	daBatch, err = NewDABatch(batch)
	assert.NoError(t, err)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 121, len(batchBytes))
	assert.Equal(t, "000000000000000000000000000000000a000000000000000ac7bcc8da943dd83404e84d9ce7e894ab97ce4829df4eb51ebbbe13c90b5a3f4d000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001dd", batchHexString)
	assert.Equal(t, 32, len(daBatch.SkippedL1MessageBitmap))
	expectedBitmap = "00000000000000000000000000000000000000000000000000000000000001dd"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(10), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(10), daBatch.L1MessagePopped)
	assert.Equal(t, common.HexToHash("0xe0950d500d47df4e9c443978682bcccfc8d50983f99ec9232067333a7d32a9d2"), daBatch.Hash())

	decodedDABatch, err = NewDABatchFromBytes(batchBytes)
	assert.NoError(t, err)
	decodedBatchBytes = decodedDABatch.Encode()
	decodedBatchHexString = hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)

	// Test case: many L1 Msgs in each of 2 bitmaps.
	chunk = &encoding.Chunk{
		Blocks: []*encoding.Block{block6},
	}
	chunkL1CommitCalldataSize, err = EstimateChunkL1CommitCalldataSize(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), chunkL1CommitCalldataSize)
	chunkL1CommitGas, err = EstimateChunkL1CommitGas(chunk)
	assert.NoError(t, err)
	assert.Equal(t, uint64(7326), chunkL1CommitGas)

	daChunk, err = NewDAChunk(chunk, 0)
	assert.NoError(t, err)
	chunkBytes, err = daChunk.Encode()
	assert.NoError(t, err)
	assert.Equal(t, 61, len(chunkBytes))
	daChunkHash, err = daChunk.Hash()
	assert.NoError(t, err)
	assert.Equal(t, common.HexToHash("0xb65521bea7daff75838de07951c3c055966750fb5a270fead5e0e727c32455c3"), daChunkHash)

	batch = &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunkHash,
		EndChunkHash:               daChunkHash,
	}

	batchL1CommitCalldataSize, err = EstimateBatchL1CommitCalldataSize(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), batchL1CommitCalldataSize)
	batchL1CommitGas, err = EstimateBatchL1CommitGas(batch)
	assert.NoError(t, err)
	assert.Equal(t, uint64(164388), batchL1CommitGas)

	daBatch, err = NewDABatch(batch)
	assert.NoError(t, err)
	batchBytes = daBatch.Encode()
	batchHexString = hex.EncodeToString(batchBytes)
	assert.Equal(t, 153, len(batchBytes))
	assert.Equal(t, "00000000000000000000000000000001010000000000000101899a411a3309c6491701b7b955c7b1115ac015414bbb71b59a0ca561668d52080000000000000000000000000000000000000000000000000000000000000000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd0000000000000000000000000000000000000000000000000000000000000000", batchHexString)
	assert.Equal(t, 64, len(daBatch.SkippedL1MessageBitmap))
	expectedBitmap = "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd0000000000000000000000000000000000000000000000000000000000000000"
	assert.Equal(t, expectedBitmap, common.Bytes2Hex(daBatch.SkippedL1MessageBitmap))
	assert.Equal(t, uint64(257), daBatch.TotalL1MessagePopped)
	assert.Equal(t, uint64(257), daBatch.L1MessagePopped)
	assert.Equal(t, common.HexToHash("0x745a74773cdc7cd0b86b50305f6373c7efeaf051b38a71ea561333708e8a90d9"), daBatch.Hash())

	decodedDABatch, err = NewDABatchFromBytes(batchBytes)
	assert.NoError(t, err)
	decodedBatchBytes = decodedDABatch.Encode()
	decodedBatchHexString = hex.EncodeToString(decodedBatchBytes)
	assert.Equal(t, batchHexString, decodedBatchHexString)
}

func readBlockFromJSON(t *testing.T, filename string) *encoding.Block {
	data, err := os.ReadFile(filename)
	assert.NoError(t, err)

	block := &encoding.Block{}
	assert.NoError(t, json.Unmarshal(data, block))
	return block
}
