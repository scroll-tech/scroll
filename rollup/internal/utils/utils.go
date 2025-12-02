package utils

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"
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
func encodeBatchHeaderValidium(b *encoding.Batch, codecVersion encoding.CodecVersion) ([]byte, common.Hash, error) {
	if b == nil {
		return nil, common.Hash{}, fmt.Errorf("batch is nil, version: %v, index: %v", codecVersion, b.Index)
	}

	if len(b.Blocks) == 0 {
		return nil, common.Hash{}, fmt.Errorf("batch contains no blocks, version: %v, index: %v", codecVersion, b.Index)
	}

	// For validium mode, use the last block hash as commitment to the off-chain data
	// TODO: This is a temporary solution, we might use a larger commitment in the future
	lastBlock := b.Blocks[len(b.Blocks)-1]
	commitment := lastBlock.Header.Hash()
	stateRoot := b.StateRoot()

	// Temporary workaround for the wrong genesis state root configuration issue.
	if lastBlock.Header.Number.Uint64() == 0 {
		if commitment == common.HexToHash("0x76a8e1359fe1a51ec3917ca98dec95ba005f1a73bcdbc2c7f87c7683e828fbb1") && stateRoot == common.HexToHash("0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339") {
			// cloak-xen/sepolia
			stateRoot = common.HexToHash("0x0711f02d6f85b0597c4705298e01ee27159fdd8bd8bdeda670ae8b9073091246")
		} else if commitment == common.HexToHash("0x8005a02271085eaded2565f3e252013cd9d3cd0a4775d89f9ba4224289671276") && stateRoot == common.HexToHash("0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339") {
			// cloak-xen/mainnet
			stateRoot = common.HexToHash("0x8da1aaf41660ddf7870ab5ff4f6a3ab4b2e652568d341ede87ada56aad5fb097")
		} else if commitment == common.HexToHash("0xa7e50dfc812039410c2009c74cdcb0c0797aa5485dec062985eaa43b17d333ea") && stateRoot == common.HexToHash("0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339") {
			// cloak-etherfi/sepolia
			stateRoot = common.HexToHash("0x7b44ea23770dda8810801779eb6847d56be0399e35de7c56465ccf8b7578ddf6")
		} else if commitment == common.HexToHash("0xeccf4fab24f8b5dd3b72667c6bf5e28b17ccffdea01e3e5c08f393edaa9e7657") && stateRoot == common.HexToHash("0x08d535cc60f40af5dd3b31e0998d7567c2d568b224bed2ba26070aeb078d1339") {
			// cloak-shiga/sepolia
			stateRoot = common.HexToHash("0x05973227854ac82c22f164ed3d4510b7df516a0eecdfd9bed5f2446efc9994b9")
		}

		log.Warn("Using genesis state root", "stateRoot", stateRoot.Hex())
	}

	// Batch header field sizes
	const (
		versionSize      = 1
		indexSize        = 8
		parentHashSize   = 32
		stateRootSize    = 32
		withdrawRootSize = 32
		commitmentSize   = 32 // TODO: 32 bytes for now, might use larger commitment in the future

		// Total size of validium batch header
		validiumBatchHeaderSize = versionSize + indexSize + parentHashSize + stateRootSize + withdrawRootSize + commitmentSize
	)

	batchBytes := make([]byte, validiumBatchHeaderSize)

	// Define offsets for each field
	var (
		versionOffset      = 0
		indexOffset        = versionOffset + versionSize
		parentHashOffset   = indexOffset + indexSize
		stateRootOffset    = parentHashOffset + parentHashSize
		withdrawRootOffset = stateRootOffset + stateRootSize
		commitmentOffset   = withdrawRootOffset + withdrawRootSize
	)

	var version uint8
	if codecVersion == encoding.CodecV8 || codecVersion == encoding.CodecV9 || codecVersion == encoding.CodecV10 {
		// Validium version line starts with v1,
		// but rollup-relayer behavior follows v8.
		version = 1
	} else if codecVersion == encoding.CodecV0 {
		// Special case for genesis batch
		version = 0
	} else {
		return nil, common.Hash{}, fmt.Errorf("unexpected codec version %d for batch %v in validium mode", codecVersion, b.Index)
	}

	batchBytes[versionOffset] = version                                                                                    // version
	binary.BigEndian.PutUint64(batchBytes[indexOffset:indexOffset+indexSize], b.Index)                                     // batch index
	copy(batchBytes[parentHashOffset:parentHashOffset+parentHashSize], b.ParentBatchHash[0:parentHashSize])                // parentBatchHash
	copy(batchBytes[stateRootOffset:stateRootOffset+stateRootSize], stateRoot.Bytes()[0:stateRootSize])                    // postStateRoot
	copy(batchBytes[withdrawRootOffset:withdrawRootOffset+withdrawRootSize], b.WithdrawRoot().Bytes()[0:withdrawRootSize]) // postWithdrawRoot
	copy(batchBytes[commitmentOffset:commitmentOffset+commitmentSize], commitment[0:commitmentSize])                       // data commitment

	hash := crypto.Keccak256Hash(batchBytes)
	return batchBytes, hash, nil
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
		batchMeta.BatchBytes, batchMeta.BatchHash, err = encodeBatchHeaderValidium(batch, codecVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to encode batch header for validium, version: %v, index: %v, err: %w", codecVersion, batch.Index, err)
		}
	}

	batchMeta.BatchBlobDataProof, err = daBatch.BlobDataProofForPointEvaluation()
	if err != nil {
		return nil, fmt.Errorf("failed to get blob data proof, version: %v, index: %v, err: %w", codecVersion, batch.Index, err)
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
