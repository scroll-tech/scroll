package utils

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func genesisBatch() *encoding.Batch {
	genesis := &gethTypes.Header{
		Number:     big.NewInt(0),
		Difficulty: big.NewInt(1),
		GasLimit:   10000000,
		Root:       common.HexToHash("0x1a4a1e5e4f9f12a88d4d38163e9ea4b7b0b52e23d72f09b56f1d49d6ad21e63a"),
	}
	chunk := &encoding.Chunk{Blocks: []*encoding.Block{{Header: genesis}}}
	return &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		Blocks:                     chunk.Blocks,
	}
}

func TestGetBatchMetadataGenesisBatch(t *testing.T) {
	// The genesis batch always uses the rollup encoding, in rollup mode and in validium mode
	// alike: a validium chain is a rollup chain migrated to ScrollChainValidiumV2 at some batch
	// height, so committedBatches[0] inherited from the rollup is the rollup genesis batch hash.
	meta, err := GetBatchMetadata(genesisBatch(), encoding.CodecV0, false)
	require.NoError(t, err)

	// The rollup CodecV0 batch header is 89 bytes:
	// version (1) + batchIndex (8) + l1MessagePopped (8) + totalL1MessagePopped (8) +
	// dataHash (32) + parentBatchHash (32).
	assert.Len(t, meta.BatchBytes, 89)
	assert.NotEqual(t, common.Hash{}, meta.BatchHash)

	// The genesis batch can no longer be encoded as a validium batch header: the validium
	// contract removes importGenesisBatch, so a validium-encoded genesis would never match
	// the on-chain genesis batch hash.
	_, err = GetBatchMetadata(genesisBatch(), encoding.CodecV0, true)
	assert.Error(t, err)
}

func TestGetBatchMetadataValidiumHeader(t *testing.T) {
	// A validium batch header is 137 bytes:
	// version (1) + batchIndex (8) + parentBatchHash (32) + postStateRoot (32) +
	// withdrawRoot (32) + commitment (32), matching BatchHeaderValidiumV0Codec with a
	// 32-byte commitment.
	header := &gethTypes.Header{
		Number:     big.NewInt(1),
		Difficulty: big.NewInt(1),
		GasLimit:   10000000,
		Root:       common.HexToHash("0x2b5b2f6f5fa023b99e5e49274f0fb5c8c1c63f34e83f0ac67f2e5ae7be32f74b"),
	}
	chunk := &encoding.Chunk{Blocks: []*encoding.Block{{Header: header}}}
	batch := &encoding.Batch{
		Index:           1,
		ParentBatchHash: common.HexToHash("0x010203"),
		Chunks:          []*encoding.Chunk{chunk},
		Blocks:          chunk.Blocks,
	}

	meta, err := GetBatchMetadata(batch, encoding.CodecV10, true)
	require.NoError(t, err)
	assert.Len(t, meta.BatchBytes, 137)

	// version 1, big-endian batch index 1
	assert.Equal(t, uint8(1), meta.BatchBytes[0])
	assert.Equal(t, batch.ParentBatchHash.Bytes(), meta.BatchBytes[9:41])
}
