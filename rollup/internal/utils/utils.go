package utils

import (
	"fmt"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
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

	codec, err := encoding.CodecFromVersion(codecVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get codec from version: %v, err: %w", codecVersion, err)
	}

	start := time.Now()
	metrics.L1CommitGas, err = codec.EstimateChunkL1CommitGas(chunk)
	metrics.EstimateGasTime = time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate chunk L1 commit gas: %w", err)
	}

	start = time.Now()
	metrics.L1CommitCalldataSize, err = codec.EstimateChunkL1CommitCalldataSize(chunk)
	metrics.EstimateCalldataSizeTime = time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate chunk L1 commit calldata size: %w", err)
	}

	start = time.Now()
	metrics.L1CommitUncompressedBatchBytesSize, metrics.L1CommitBlobSize, err = codec.EstimateChunkL1CommitBatchSizeAndBlobSize(chunk)
	metrics.EstimateBlobSizeTime = time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate chunk L1 commit batch size and blob size: %w", err)
	}
	return metrics, nil
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

	codec, err := encoding.CodecFromVersion(codecVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get codec from version: %v, err: %w", codecVersion, err)
	}

	start := time.Now()
	metrics.L1CommitGas, err = codec.EstimateBatchL1CommitGas(batch)
	metrics.EstimateGasTime = time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate batch L1 commit gas: %w", err)
	}

	metrics.L1CommitCalldataSize, err = codec.EstimateBatchL1CommitCalldataSize(batch)
	metrics.EstimateCalldataSizeTime = time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate batch L1 commit calldata size: %w", err)
	}

	start = time.Now()
	metrics.L1CommitUncompressedBatchBytesSize, metrics.L1CommitBlobSize, err = codec.EstimateBatchL1CommitBatchSizeAndBlobSize(batch)
	metrics.EstimateBlobSizeTime = time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate batch L1 commit batch size and blob size: %w", err)
	}
	return metrics, nil
}

// GetChunkHash retrieves the hash of a chunk.
func GetChunkHash(chunk *encoding.Chunk, totalL1MessagePoppedBefore uint64, codecVersion encoding.CodecVersion) (common.Hash, error) {
	codec, err := encoding.CodecFromVersion(codecVersion)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get codec from version: %v, err: %w", codecVersion, err)
	}

	daChunk, err := codec.NewDAChunk(chunk, totalL1MessagePoppedBefore)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create DA chunk: %w", err)
	}

	chunkHash, err := daChunk.Hash()
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get DA chunk hash: %w", err)
	}

	return chunkHash, nil
}

// BatchMetadata represents the metadata of a batch.
type BatchMetadata struct {
	BatchHash          common.Hash
	BatchDataHash      common.Hash
	BatchBlobDataProof []byte
	BatchBytes         []byte
	StartChunkHash     common.Hash
	EndChunkHash       common.Hash
	BlobBytes          []byte
}

// GetBatchMetadata retrieves the metadata of a batch.
func GetBatchMetadata(batch *encoding.Batch, codecVersion encoding.CodecVersion) (*BatchMetadata, error) {
	codec, err := encoding.CodecFromVersion(codecVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get codec from version: %v, err: %w", codecVersion, err)
	}

	daBatch, err := codec.NewDABatch(batch)
	if err != nil {
		return nil, fmt.Errorf("failed to create DA batch: %w", err)
	}

	batchMeta := &BatchMetadata{
		BatchHash:     daBatch.Hash(),
		BatchDataHash: daBatch.DataHash(),
		BatchBytes:    daBatch.Encode(),
		BlobBytes:     daBatch.BlobBytes(),
	}

	batchMeta.BatchBlobDataProof, err = daBatch.BlobDataProofForPointEvaluation()
	if err != nil {
		return nil, fmt.Errorf("failed to get blob data proof: %w", err)
	}

	numChunks := len(batch.Chunks)
	if numChunks == 0 {
		return nil, fmt.Errorf("batch contains no chunks")
	}

	startDAChunk, err := codec.NewDAChunk(batch.Chunks[0], batch.TotalL1MessagePoppedBefore)
	if err != nil {
		return nil, fmt.Errorf("failed to create start DA chunk: %w", err)
	}

	batchMeta.StartChunkHash, err = startDAChunk.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get start DA chunk hash: %w", err)
	}

	var totalL1MessagePoppedBeforeEndDAChunk uint64
	for i := 0; i < len(batch.Chunks)-1; i++ {
		totalL1MessagePoppedBeforeEndDAChunk += batch.Chunks[i].NumL1Messages(totalL1MessagePoppedBeforeEndDAChunk)
	}
	endDAChunk, err := codec.NewDAChunk(batch.Chunks[numChunks-1], totalL1MessagePoppedBeforeEndDAChunk)
	if err != nil {
		return nil, fmt.Errorf("failed to create end DA chunk: %w", err)
	}

	batchMeta.EndChunkHash, err = endDAChunk.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get end DA chunk hash: %w", err)
	}

	return batchMeta, nil
}
