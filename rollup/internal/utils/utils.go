package utils

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
)

// ChunkMetrics indicates the metrics for proposing a chunk.
type ChunkMetrics struct {
	NumBlocks           uint64
	TxNum               uint64
	L2Gas               uint64
	FirstBlockTimestamp uint64

	L1CommitBlobSize                   uint64
	L1CommitUncompressedBatchBytesSize uint64

	// timing metrics
	EstimateBlobSizeTime time.Duration
}

// CalculateChunkMetrics calculates chunk metrics.
func CalculateChunkMetrics(chunk *encoding.Chunk, codecVersion encoding.CodecVersion) (*ChunkMetrics, error) {
	metrics := &ChunkMetrics{
		TxNum:               chunk.NumTransactions(),
		NumBlocks:           uint64(len(chunk.Blocks)),
		FirstBlockTimestamp: chunk.Blocks[0].Header.Time,
	}

	// Get total L2 gas for chunk
	for _, block := range chunk.Blocks {
		metrics.L2Gas += block.Header.GasUsed
	}

	var err error
	codec, err := encoding.CodecFromVersion(codecVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get codec from version: %v, err: %w", codecVersion, err)
	}

	metrics.EstimateBlobSizeTime, err = measureTime(func() error {
		metrics.L1CommitUncompressedBatchBytesSize, metrics.L1CommitBlobSize, err = codec.EstimateChunkL1CommitBatchSizeAndBlobSize(chunk)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to estimate chunk L1 commit batch size and blob size, version: %v, err: %w", codecVersion, err)
	}

	return metrics, nil
}

// BatchMetrics indicates the metrics for proposing a batch.
type BatchMetrics struct {
	NumChunks           uint64
	FirstBlockTimestamp uint64

	L1CommitBlobSize                   uint64
	L1CommitUncompressedBatchBytesSize uint64

	ValidiumMode bool // default false: rollup mode

	// timing metrics
	EstimateBlobSizeTime time.Duration
}

// CalculateBatchMetrics calculates batch metrics.
func CalculateBatchMetrics(batch *encoding.Batch, codecVersion encoding.CodecVersion, validiumMode bool) (*BatchMetrics, error) {
	metrics := &BatchMetrics{
		NumChunks:           uint64(len(batch.Chunks)),
		FirstBlockTimestamp: batch.Chunks[0].Blocks[0].Header.Time,
		ValidiumMode:        validiumMode,
	}

	codec, err := encoding.CodecFromVersion(codecVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get codec from version: %v, err: %w", codecVersion, err)
	}

	metrics.EstimateBlobSizeTime, err = measureTime(func() error {
		metrics.L1CommitUncompressedBatchBytesSize, metrics.L1CommitBlobSize, err = codec.EstimateBatchL1CommitBatchSizeAndBlobSize(batch)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to estimate batch L1 commit batch size and blob size, version: %v, err: %w", codecVersion, err)
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
		return common.Hash{}, fmt.Errorf("failed to create DA chunk, version: %v, err: %w", codecVersion, err)
	}

	chunkHash, err := daChunk.Hash()
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get DA chunk hash, version: %v, err: %w", codecVersion, err)
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
	ChallengeDigest    common.Hash
}

// encodeBatchHeaderValidium encodes batch header for validium mode and returns both encoded bytes and hash
func encodeBatchHeaderValidium(b *encoding.Batch, codecVersion encoding.CodecVersion) ([]byte, common.Hash) {
	batchBytes := make([]byte, 105+32)                       // todo: commitment
	batchBytes[0] = uint8(codecVersion)                      // version
	binary.BigEndian.PutUint64(batchBytes[1:9], b.Index)     // batch index
	copy(batchBytes[9:41], b.ParentBatchHash[0:32])          // parentBatchHash
	copy(batchBytes[41:73], b.StateRoot().Bytes()[0:32])     // postStateRoot
	copy(batchBytes[73:105], b.WithdrawRoot().Bytes()[0:32]) // postWithdrawRoot

	var commitment common.Hash
	if len(b.Blocks) > 0 {
		lastBlock := b.Blocks[len(b.Blocks)-1]
		commitment = lastBlock.Header.Hash()
	}
	copy(batchBytes[105:137], commitment[0:32]) // data commitment

	hash := crypto.Keccak256Hash(batchBytes)
	return batchBytes, hash
}

// GetBatchMetadata retrieves the metadata of a batch.
func GetBatchMetadata(batch *encoding.Batch, codecVersion encoding.CodecVersion, validiumMode bool) (*BatchMetadata, error) {
	codec, err := encoding.CodecFromVersion(codecVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get codec from version: %v, err: %w", codecVersion, err)
	}

	daBatch, err := codec.NewDABatch(batch)
	if err != nil {
		return nil, fmt.Errorf("failed to create DA batch, version: %v, err: %w", codecVersion, err)
	}

	batchMeta := &BatchMetadata{
		BatchHash:       daBatch.Hash(),
		BatchDataHash:   daBatch.DataHash(),
		BatchBytes:      daBatch.Encode(),
		BlobBytes:       daBatch.BlobBytes(),
		ChallengeDigest: daBatch.ChallengeDigest(),
	}

	// If this function is used in Validium, we encode the batch header differently.
	if validiumMode {
		batchMeta.BatchBytes, batchMeta.BatchHash = encodeBatchHeaderValidium(batch, codecVersion)
	}

	batchMeta.BatchBlobDataProof, err = daBatch.BlobDataProofForPointEvaluation()
	if err != nil {
		return nil, fmt.Errorf("failed to get blob data proof, version: %v, err: %w", codecVersion, err)
	}

	numChunks := len(batch.Chunks)
	if numChunks == 0 {
		return nil, fmt.Errorf("batch contains no chunks, version: %v, index: %v", codecVersion, batch.Index)
	}

	startDAChunk, err := codec.NewDAChunk(batch.Chunks[0], batch.TotalL1MessagePoppedBefore)
	if err != nil {
		return nil, fmt.Errorf("failed to create start DA chunk, version: %v, err: %w", codecVersion, err)
	}

	batchMeta.StartChunkHash, err = startDAChunk.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get start DA chunk hash, version: %v, err: %w", codecVersion, err)
	}

	totalL1MessagePoppedBeforeEndDAChunk := batch.TotalL1MessagePoppedBefore
	for i := 0; i < len(batch.Chunks)-1; i++ {
		totalL1MessagePoppedBeforeEndDAChunk += batch.Chunks[i].NumL1Messages(totalL1MessagePoppedBeforeEndDAChunk)
	}
	endDAChunk, err := codec.NewDAChunk(batch.Chunks[numChunks-1], totalL1MessagePoppedBeforeEndDAChunk)
	if err != nil {
		return nil, fmt.Errorf("failed to create end DA chunk, version: %v, err: %w", codecVersion, err)
	}

	batchMeta.EndChunkHash, err = endDAChunk.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to get end DA chunk hash, version: %v, err: %w", codecVersion, err)
	}

	return batchMeta, nil
}

func measureTime(operation func() error) (time.Duration, error) {
	start := time.Now()
	err := operation()
	return time.Since(start), err
}
