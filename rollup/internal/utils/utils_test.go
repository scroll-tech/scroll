package utils

import (
	"errors"
	"math/big"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/da-codec/encoding/codecv2"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

// regression test
func TestCompressedDataCompatibilityErrorCatching(t *testing.T) {
	block := &encoding.Block{
		Header: &types.Header{
			Number: big.NewInt(0),
		},
		RowConsumption: &types.RowConsumption{},
	}
	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{block},
	}
	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
	}

	patchGuard1 := gomonkey.ApplyFunc(codecv2.EstimateChunkL1CommitBatchSizeAndBlobSize, func(b *encoding.Chunk) (uint64, uint64, error) {
		return 0, 0, &encoding.CompressedDataCompatibilityError{Err: errors.New("test-error-1")}
	})
	defer patchGuard1.Reset()

	var compressErr *encoding.CompressedDataCompatibilityError

	_, err := CalculateChunkMetrics(chunk, encoding.CodecV2)
	assert.Error(t, err)
	assert.ErrorAs(t, err, &compressErr)

	patchGuard2 := gomonkey.ApplyFunc(codecv2.EstimateBatchL1CommitBatchSizeAndBlobSize, func(b *encoding.Batch) (uint64, uint64, error) {
		return 0, 0, &encoding.CompressedDataCompatibilityError{Err: errors.New("test-error-2")}
	})
	defer patchGuard2.Reset()

	_, err = CalculateBatchMetrics(batch, encoding.CodecV2)
	assert.Error(t, err)
	assert.ErrorAs(t, err, &compressErr)
}
