package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/da-codec/encoding/codecv0"
	"github.com/scroll-tech/da-codec/encoding/codecv1"
	"github.com/scroll-tech/da-codec/encoding/codecv2"
	"github.com/scroll-tech/go-ethereum/common"
)

// ChunkMetrics indicates the metrics for proposing a chunk.
type ChunkMetrics struct {
	// common metrics
	NumBlocks           uint64
	TxNum               uint64
	CrcMax              uint64
	FirstBlockTimestamp uint64

	L1CommitCalldataSize uint64
	L1CommitGas          uint64

	// codecv1 metrics, default 0 for codecv0
	L1CommitBlobSize uint64

	// codecv2 metrics, default 0 for codecv0 & codecv1
	L1CommitUncompressedBatchBytesSize uint64

	// timing metrics
	EstimateGasTime          time.Duration
	EstimateCalldataSizeTime time.Duration
	EstimateBlobSizeTime     time.Duration
}

// CalculateChunkMetrics calculates chunk metrics.
func CalculateChunkMetrics(chunk *encoding.Chunk, codecVersion encoding.CodecVersion) (*ChunkMetrics, error) {
	var err error
	metrics := &ChunkMetrics{
		TxNum:               chunk.NumTransactions(),
		NumBlocks:           uint64(len(chunk.Blocks)),
		FirstBlockTimestamp: chunk.Blocks[0].Header.Time,
	}
	metrics.CrcMax, err = chunk.CrcMax()
	if err != nil {
		return nil, fmt.Errorf("failed to get crc max: %w", err)
	}
	switch codecVersion {
	case encoding.CodecV0:
		start := time.Now()
		metrics.L1CommitGas, err = codecv0.EstimateChunkL1CommitGas(chunk)
		metrics.EstimateGasTime = time.Since(start)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate codecv0 chunk L1 commit gas: %w", err)
		}

		start = time.Now()
		metrics.L1CommitCalldataSize, err = codecv0.EstimateChunkL1CommitCalldataSize(chunk)
		metrics.EstimateCalldataSizeTime = time.Since(start)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate codecv0 chunk L1 commit calldata size: %w", err)
		}
		return metrics, nil
	case encoding.CodecV1:
		start := time.Now()
		metrics.L1CommitGas = codecv1.EstimateChunkL1CommitGas(chunk)
		metrics.EstimateGasTime = time.Since(start)

		start = time.Now()
		metrics.L1CommitCalldataSize = codecv1.EstimateChunkL1CommitCalldataSize(chunk)
		metrics.EstimateCalldataSizeTime = time.Since(start)

		start = time.Now()
		metrics.L1CommitBlobSize, err = codecv1.EstimateChunkL1CommitBlobSize(chunk)
		metrics.EstimateBlobSizeTime = time.Since(start)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate codecv1 chunk L1 commit blob size: %w", err)
		}
		return metrics, nil
	case encoding.CodecV2:
		start := time.Now()
		metrics.L1CommitGas = codecv2.EstimateChunkL1CommitGas(chunk)
		metrics.EstimateGasTime = time.Since(start)

		start = time.Now()
		metrics.L1CommitCalldataSize = codecv2.EstimateChunkL1CommitCalldataSize(chunk)
		metrics.EstimateCalldataSizeTime = time.Since(start)

		start = time.Now()
		metrics.L1CommitUncompressedBatchBytesSize, metrics.L1CommitBlobSize, err = codecv2.EstimateChunkL1CommitBatchSizeAndBlobSize(chunk)
		metrics.EstimateBlobSizeTime = time.Since(start)
		if err != nil {
			var compressErr *encoding.CompressedDataCompatibilityError
			if errors.As(err, &compressErr) {
				return nil, err
			} else {
				return nil, fmt.Errorf("failed to estimate codecv2 chunk L1 commit batch size and blob size: %w", err)
			}
		}
		return metrics, nil
	default:
		return nil, fmt.Errorf("unsupported codec version: %v", codecVersion)
	}
}

// BatchMetrics indicates the metrics for proposing a batch.
type BatchMetrics struct {
	// common metrics
	NumChunks           uint64
	FirstBlockTimestamp uint64

	L1CommitCalldataSize uint64
	L1CommitGas          uint64

	// codecv1 metrics, default 0 for codecv0
	L1CommitBlobSize uint64

	// codecv2 metrics, default 0 for codecv0 & codecv1
	L1CommitUncompressedBatchBytesSize uint64

	// timing metrics
	EstimateGasTime          time.Duration
	EstimateCalldataSizeTime time.Duration
	EstimateBlobSizeTime     time.Duration
}

// CalculateBatchMetrics calculates batch metrics.
func CalculateBatchMetrics(batch *encoding.Batch, codecVersion encoding.CodecVersion) (*BatchMetrics, error) {
	var err error
	metrics := &BatchMetrics{
		NumChunks:           uint64(len(batch.Chunks)),
		FirstBlockTimestamp: batch.Chunks[0].Blocks[0].Header.Time,
	}
	switch codecVersion {
	case encoding.CodecV0:
		start := time.Now()
		metrics.L1CommitGas, err = codecv0.EstimateBatchL1CommitGas(batch)
		metrics.EstimateGasTime = time.Since(start)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate codecv0 batch L1 commit gas: %w", err)
		}
		start = time.Now()
		metrics.L1CommitCalldataSize, err = codecv0.EstimateBatchL1CommitCalldataSize(batch)
		metrics.EstimateCalldataSizeTime = time.Since(start)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate codecv0 batch L1 commit calldata size: %w", err)
		}
		return metrics, nil
	case encoding.CodecV1:
		start := time.Now()
		metrics.L1CommitGas = codecv1.EstimateBatchL1CommitGas(batch)
		metrics.EstimateGasTime = time.Since(start)

		start = time.Now()
		metrics.L1CommitCalldataSize = codecv1.EstimateBatchL1CommitCalldataSize(batch)
		metrics.EstimateCalldataSizeTime = time.Since(start)

		start = time.Now()
		metrics.L1CommitBlobSize, err = codecv1.EstimateBatchL1CommitBlobSize(batch)
		metrics.EstimateBlobSizeTime = time.Since(start)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate codecv1 batch L1 commit blob size: %w", err)
		}
		return metrics, nil
	case encoding.CodecV2:
		start := time.Now()
		metrics.L1CommitGas = codecv2.EstimateBatchL1CommitGas(batch)
		metrics.EstimateGasTime = time.Since(start)

		start = time.Now()
		metrics.L1CommitCalldataSize = codecv2.EstimateBatchL1CommitCalldataSize(batch)
		metrics.EstimateCalldataSizeTime = time.Since(start)

		start = time.Now()
		metrics.L1CommitUncompressedBatchBytesSize, metrics.L1CommitBlobSize, err = codecv2.EstimateBatchL1CommitBatchSizeAndBlobSize(batch)
		metrics.EstimateBlobSizeTime = time.Since(start)
		if err != nil {
			var compressErr *encoding.CompressedDataCompatibilityError
			if errors.As(err, &compressErr) {
				return nil, err
			} else {
				return nil, fmt.Errorf("failed to estimate codecv2 batch L1 commit batch size and blob size: %w", err)
			}
		}
		return metrics, nil
	default:
		return nil, fmt.Errorf("unsupported codec version: %v", codecVersion)
	}
}

// GetChunkHash retrieves the hash of a chunk.
func GetChunkHash(chunk *encoding.Chunk, totalL1MessagePoppedBefore uint64, codecVersion encoding.CodecVersion) (common.Hash, error) {
	switch codecVersion {
	case encoding.CodecV0:
		daChunk, err := codecv0.NewDAChunk(chunk, totalL1MessagePoppedBefore)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to create codecv0 DA chunk: %w", err)
		}
		chunkHash, err := daChunk.Hash()
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to get codecv0 DA chunk hash: %w", err)
		}
		return chunkHash, nil
	case encoding.CodecV1:
		daChunk, err := codecv1.NewDAChunk(chunk, totalL1MessagePoppedBefore)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to create codecv1 DA chunk: %w", err)
		}
		chunkHash, err := daChunk.Hash()
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to get codecv1 DA chunk hash: %w", err)
		}
		return chunkHash, nil
	case encoding.CodecV2:
		daChunk, err := codecv2.NewDAChunk(chunk, totalL1MessagePoppedBefore)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to create codecv2 DA chunk: %w", err)
		}
		chunkHash, err := daChunk.Hash()
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to get codecv2 DA chunk hash: %w", err)
		}
		return chunkHash, nil
	default:
		return common.Hash{}, fmt.Errorf("unsupported codec version: %v", codecVersion)
	}
}

// BatchMetadata represents the metadata of a batch.
type BatchMetadata struct {
	BatchHash          common.Hash
	BatchDataHash      common.Hash
	BatchBlobDataProof []byte
	BatchBytes         []byte
	StartChunkHash     common.Hash
	EndChunkHash       common.Hash
}

// GetBatchMetadata retrieves the metadata of a batch.
func GetBatchMetadata(batch *encoding.Batch, codecVersion encoding.CodecVersion) (*BatchMetadata, error) {
	numChunks := len(batch.Chunks)
	totalL1MessagePoppedBeforeEndDAChunk := batch.TotalL1MessagePoppedBefore
	for i := 0; i < numChunks-1; i++ {
		totalL1MessagePoppedBeforeEndDAChunk += batch.Chunks[i].NumL1Messages(totalL1MessagePoppedBeforeEndDAChunk)
	}

	switch codecVersion {
	case encoding.CodecV0:
		daBatch, err := codecv0.NewDABatch(batch)
		if err != nil {
			return nil, fmt.Errorf("failed to create codecv0 DA batch: %w", err)
		}

		// BatchBlobDataProof is left as empty for codecv0.
		batchMeta := &BatchMetadata{
			BatchHash:     daBatch.Hash(),
			BatchDataHash: daBatch.DataHash,
			BatchBytes:    daBatch.Encode(),
		}

		startDAChunk, err := codecv0.NewDAChunk(batch.Chunks[0], batch.TotalL1MessagePoppedBefore)
		if err != nil {
			return nil, fmt.Errorf("failed to create codecv0 start DA chunk: %w", err)
		}

		batchMeta.StartChunkHash, err = startDAChunk.Hash()
		if err != nil {
			return nil, fmt.Errorf("failed to get codecv0 start DA chunk hash: %w", err)
		}

		endDAChunk, err := codecv0.NewDAChunk(batch.Chunks[numChunks-1], totalL1MessagePoppedBeforeEndDAChunk)
		if err != nil {
			return nil, fmt.Errorf("failed to create codecv0 end DA chunk: %w", err)
		}

		batchMeta.EndChunkHash, err = endDAChunk.Hash()
		if err != nil {
			return nil, fmt.Errorf("failed to get codecv0 end DA chunk hash: %w", err)
		}
		return batchMeta, nil
	case encoding.CodecV1:
		daBatch, err := codecv1.NewDABatch(batch)
		if err != nil {
			return nil, fmt.Errorf("failed to create codecv1 DA batch: %w", err)
		}

		blobDataProof, err := daBatch.BlobDataProof()
		if err != nil {
			return nil, fmt.Errorf("failed to get codecv1 blob data proof: %w", err)
		}

		batchMeta := &BatchMetadata{
			BatchHash:          daBatch.Hash(),
			BatchDataHash:      daBatch.DataHash,
			BatchBlobDataProof: blobDataProof,
			BatchBytes:         daBatch.Encode(),
		}

		startDAChunk, err := codecv1.NewDAChunk(batch.Chunks[0], batch.TotalL1MessagePoppedBefore)
		if err != nil {
			return nil, fmt.Errorf("failed to create codecv1 start DA chunk: %w", err)
		}

		batchMeta.StartChunkHash, err = startDAChunk.Hash()
		if err != nil {
			return nil, fmt.Errorf("failed to get codecv1 start DA chunk hash: %w", err)
		}

		endDAChunk, err := codecv1.NewDAChunk(batch.Chunks[numChunks-1], totalL1MessagePoppedBeforeEndDAChunk)
		if err != nil {
			return nil, fmt.Errorf("failed to create codecv1 end DA chunk: %w", err)
		}

		batchMeta.EndChunkHash, err = endDAChunk.Hash()
		if err != nil {
			return nil, fmt.Errorf("failed to get codecv1 end DA chunk hash: %w", err)
		}
		return batchMeta, nil
	case encoding.CodecV2:
		daBatch, err := codecv2.NewDABatch(batch)
		if err != nil {
			return nil, fmt.Errorf("failed to create codecv2 DA batch: %w", err)
		}

		blobDataProof, err := daBatch.BlobDataProof()
		if err != nil {
			return nil, fmt.Errorf("failed to get codecv2 blob data proof: %w", err)
		}

		batchMeta := &BatchMetadata{
			BatchHash:          daBatch.Hash(),
			BatchDataHash:      daBatch.DataHash,
			BatchBlobDataProof: blobDataProof,
			BatchBytes:         daBatch.Encode(),
		}

		startDAChunk, err := codecv2.NewDAChunk(batch.Chunks[0], batch.TotalL1MessagePoppedBefore)
		if err != nil {
			return nil, fmt.Errorf("failed to create codecv2 start DA chunk: %w", err)
		}

		batchMeta.StartChunkHash, err = startDAChunk.Hash()
		if err != nil {
			return nil, fmt.Errorf("failed to get codecv2 start DA chunk hash: %w", err)
		}

		endDAChunk, err := codecv2.NewDAChunk(batch.Chunks[numChunks-1], totalL1MessagePoppedBeforeEndDAChunk)
		if err != nil {
			return nil, fmt.Errorf("failed to create codecv2 end DA chunk: %w", err)
		}

		batchMeta.EndChunkHash, err = endDAChunk.Hash()
		if err != nil {
			return nil, fmt.Errorf("failed to get codecv2 end DA chunk hash: %w", err)
		}
		return batchMeta, nil
	default:
		return nil, fmt.Errorf("unsupported codec version: %v", codecVersion)
	}
}
