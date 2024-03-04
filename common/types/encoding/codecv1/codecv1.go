package codecv1

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types/encoding"
)

var BLS_MODULUS *big.Int

func init() {
	modulus, success := new(big.Int).SetString("52435875175126190479447740508185965837690552500527637822603658699938581184513", 10)
	if !success {
		log.Crit("BLS_MODULUS conversion failed")
	}
	BLS_MODULUS = modulus
}

// CodecV0Version denotes the version of the codec.
const CodecV1Version = 1

// DABlock represents a Data Availability Block.
type DABlock struct {
	BlockNumber     uint64
	Timestamp       uint64
	BaseFee         *big.Int
	GasLimit        uint64
	NumTransactions uint16
	NumL1Messages   uint16
}

// DAChunk groups consecutive DABlocks with their transactions.
type DAChunk struct {
	Blocks       []*DABlock
	Transactions [][]*types.TransactionData
}

// DABatch contains metadata about a batch of DAChunks.
type DABatch struct {
	// header
	Version                uint8
	BatchIndex             uint64
	L1MessagePopped        uint64
	TotalL1MessagePopped   uint64
	DataHash               common.Hash
	BlobVersionedHash      common.Hash
	ParentBatchHash        common.Hash
	SkippedL1MessageBitmap []byte

	// blob payload
	blob *kzg4844.Blob
	z    *kzg4844.Point
}

// NewDABlock creates a new DABlock from the given encoding.Block and the total number of L1 messages popped before.
func NewDABlock(block *encoding.Block, totalL1MessagePoppedBefore uint64) (*DABlock, error) {
	if !block.Header.Number.IsUint64() {
		return nil, errors.New("block number is not uint64")
	}

	// note: numL1Messages includes skipped messages
	numL1Messages := block.NumL1Messages(totalL1MessagePoppedBefore)
	if numL1Messages > math.MaxUint16 {
		return nil, errors.New("number of L1 messages exceeds max uint16")
	}

	// note: numTransactions includes skipped messages
	numL2Transactions := block.NumL2Transactions()
	numTransactions := numL1Messages + numL2Transactions
	if numTransactions > math.MaxUint16 {
		return nil, errors.New("number of transactions exceeds max uint16")
	}

	daBlock := DABlock{
		BlockNumber:     block.Header.Number.Uint64(),
		Timestamp:       block.Header.Time,
		BaseFee:         block.Header.BaseFee,
		GasLimit:        block.Header.GasLimit,
		NumTransactions: uint16(numTransactions),
		NumL1Messages:   uint16(numL1Messages),
	}

	return &daBlock, nil
}

// Encode serializes the DABlock into a slice of bytes.
func (b *DABlock) Encode() []byte {
	bytes := make([]byte, 60)
	binary.BigEndian.PutUint64(bytes[0:], b.BlockNumber)
	binary.BigEndian.PutUint64(bytes[8:], b.Timestamp)
	if b.BaseFee != nil {
		binary.BigEndian.PutUint64(bytes[40:], b.BaseFee.Uint64())
	}
	binary.BigEndian.PutUint64(bytes[48:], b.GasLimit)
	binary.BigEndian.PutUint16(bytes[56:], b.NumTransactions)
	binary.BigEndian.PutUint16(bytes[58:], b.NumL1Messages)
	return bytes
}

// NewDAChunk creates a new DAChunk from the given encoding.Chunk and the total number of L1 messages popped before.
func NewDAChunk(chunk *encoding.Chunk, totalL1MessagePoppedBefore uint64) (*DAChunk, error) {
	var blocks []*DABlock
	var txs [][]*types.TransactionData

	for _, block := range chunk.Blocks {
		b, err := NewDABlock(block, totalL1MessagePoppedBefore)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
		totalL1MessagePoppedBefore += block.NumL1Messages(totalL1MessagePoppedBefore)
		txs = append(txs, block.Transactions)
	}

	daChunk := DAChunk{
		Blocks:       blocks,
		Transactions: txs,
	}

	return &daChunk, nil
}

// Encode serializes the DAChunk into a slice of bytes.
func (c *DAChunk) Encode() []byte {
	var chunkBytes []byte
	chunkBytes = append(chunkBytes, byte(len(c.Blocks)))

	for _, block := range c.Blocks {
		blockBytes := block.Encode()
		chunkBytes = append(chunkBytes, blockBytes...)
	}

	return chunkBytes
}

// Hash computes the hash of the DAChunk data.
func (c *DAChunk) Hash() (common.Hash, error) {
	var dataBytes []byte

	// concatenate block contexts
	for _, block := range c.Blocks {
		encodedBlock := block.Encode()
		// only the first 58 bytes are used in the hashing process
		dataBytes = append(dataBytes, encodedBlock[:58]...)
	}

	// concatenate l1 tx hashes
	for _, blockTxs := range c.Transactions {
		for _, txData := range blockTxs {
			if txData.Type == types.L1MessageTxType {
				txHash := strings.TrimPrefix(txData.TxHash, "0x")
				hashBytes, err := hex.DecodeString(txHash)
				if err != nil {
					return common.Hash{}, err
				}
				dataBytes = append(dataBytes, hashBytes...)
			}
		}
	}

	hash := crypto.Keccak256Hash(dataBytes)
	return hash, nil
}

// NewDABatch creates a DABatch from the provided encoding.Batch.
func NewDABatch(batch *encoding.Batch) (*DABatch, error) {
	// this encoding can only support up to 15 chunks per batch
	if len(batch.Chunks) > 15 {
		return nil, fmt.Errorf("too many chunks in batch")
	}

	// batch data hash
	dataHash, err := computeBatchDataHash(batch.Chunks, batch.TotalL1MessagePoppedBefore)
	if err != nil {
		return nil, err
	}

	// skipped L1 messages bitmap
	bitmapBytes, totalL1MessagePoppedAfter, err := constructSkippedBitmap(batch.Index, batch.Chunks, batch.TotalL1MessagePoppedBefore)
	if err != nil {
		return nil, err
	}

	// blob payload
	blob, z, err := constructBlobPayload(batch.Chunks)
	if err != nil {
		return nil, err
	}

	// blob versioned hash
	c, err := kzg4844.BlobToCommitment(*blob)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob commitment")
	}
	blobVersionedHash := kzg4844.CalcBlobHashV1(sha256.New(), &c)

	daBatch := DABatch{
		Version:                CodecV1Version,
		BatchIndex:             batch.Index,
		L1MessagePopped:        totalL1MessagePoppedAfter - batch.TotalL1MessagePoppedBefore,
		TotalL1MessagePopped:   totalL1MessagePoppedAfter,
		DataHash:               dataHash,
		BlobVersionedHash:      blobVersionedHash,
		ParentBatchHash:        batch.ParentBatchHash,
		SkippedL1MessageBitmap: bitmapBytes,
		blob:                   blob,
		z:                      z,
	}

	return &daBatch, nil
}

// computeBatchDataHash computes the data hash of the batch.
// Note: The batch hash and batch data hash are two different hashes,
// the former is used for identifying a badge in the contracts,
// the latter is used in the public input to the provers.
func computeBatchDataHash(chunks []*encoding.Chunk, totalL1MessagePoppedBefore uint64) (common.Hash, error) {
	var dataBytes []byte
	totalL1MessagePoppedBeforeChunk := totalL1MessagePoppedBefore

	for _, chunk := range chunks {
		daChunk, err := NewDAChunk(chunk, totalL1MessagePoppedBeforeChunk)
		if err != nil {
			return common.Hash{}, err
		}
		totalL1MessagePoppedBeforeChunk += chunk.NumL1Messages(totalL1MessagePoppedBeforeChunk)
		chunkHash, err := daChunk.Hash()
		if err != nil {
			return common.Hash{}, err
		}
		dataBytes = append(dataBytes, chunkHash.Bytes()...)
	}

	dataHash := crypto.Keccak256Hash(dataBytes)
	return dataHash, nil
}

// constructSkippedBitmap constructs skipped L1 message bitmap of the batch.
func constructSkippedBitmap(batchIndex uint64, chunks []*encoding.Chunk, totalL1MessagePoppedBefore uint64) ([]byte, uint64, error) {
	// skipped L1 message bitmap, an array of 256-bit bitmaps
	var skippedBitmap []*big.Int

	// the first queue index that belongs to this batch
	baseIndex := totalL1MessagePoppedBefore

	// the next queue index that we need to process
	nextIndex := totalL1MessagePoppedBefore

	for chunkID, chunk := range chunks {
		for blockID, block := range chunk.Blocks {
			for _, tx := range block.Transactions {
				if tx.Type != types.L1MessageTxType {
					continue
				}
				currentIndex := tx.Nonce

				if currentIndex < nextIndex {
					return nil, 0, fmt.Errorf("unexpected batch payload, expected queue index: %d, got: %d. Batch index: %d, chunk index in batch: %d, block index in chunk: %d, block hash: %v, transaction hash: %v", nextIndex, currentIndex, batchIndex, chunkID, blockID, block.Header.Hash(), tx.TxHash)
				}

				// mark skipped messages
				for skippedIndex := nextIndex; skippedIndex < currentIndex; skippedIndex++ {
					quo := int((skippedIndex - baseIndex) / 256)
					rem := int((skippedIndex - baseIndex) % 256)
					for len(skippedBitmap) <= quo {
						bitmap := big.NewInt(0)
						skippedBitmap = append(skippedBitmap, bitmap)
					}
					skippedBitmap[quo].SetBit(skippedBitmap[quo], rem, 1)
				}

				// process included message
				quo := int((currentIndex - baseIndex) / 256)
				for len(skippedBitmap) <= quo {
					bitmap := big.NewInt(0)
					skippedBitmap = append(skippedBitmap, bitmap)
				}

				nextIndex = currentIndex + 1
			}
		}
	}

	bitmapBytes := make([]byte, len(skippedBitmap)*32)
	for ii, num := range skippedBitmap {
		bytes := num.Bytes()
		padding := 32 - len(bytes)
		copy(bitmapBytes[32*ii+padding:], bytes)
	}

	return bitmapBytes, nextIndex, nil
}

// constructSkippedBitmap constructs the 4844 blob payload.
func constructBlobPayload(chunks []*encoding.Chunk) (*kzg4844.Blob, *kzg4844.Point, error) {
	// the raw (un-padded) blob payload
	blobBytes := make([]byte, 2*31)

	// the canonical (padded) blob payload
	var blob kzg4844.Blob

	// the number of chunks that contain at least one L2 transaction
	numNonEmptyChunks := 0

	// challenge digest preimage
	// 1 hash for metadata and 1 for each chunk
	challengePreimage := make([]byte, 16*32)

	// the challenge point z
	var z kzg4844.Point

	// encode blob metadata and L2 transactions,
	// and simultaneously also build challenge preimage
	for chunkID, chunk := range chunks {
		currentChunkStartIndex := len(blobBytes)
		hasL2Tx := false

		for _, block := range chunk.Blocks {
			for _, tx := range block.Transactions {
				if tx.Type != types.L1MessageTxType {
					hasL2Tx = true
					// encode L2 txs into blob payload
					rlpTxData, err := encoding.ConvertTxDataToRLPEncoding(tx)
					if err != nil {
						return nil, nil, err
					}
					blobBytes = append(blobBytes, rlpTxData...)
					continue
				}
			}
		}

		// blob metadata: chunki_size
		chunkSize := len(blobBytes) - currentChunkStartIndex
		binary.BigEndian.PutUint32(blobBytes[2+4*chunkID:], uint32(chunkSize))

		if hasL2Tx {
			numNonEmptyChunks += 1
		}

		// challenge: compute chunk data hash
		hash := crypto.Keccak256Hash(blobBytes[currentChunkStartIndex:])
		copy(challengePreimage[32+chunkID*32:], hash[:])
	}

	// blob metadata: num_chunks
	binary.BigEndian.PutUint16(blobBytes[0:], uint16(numNonEmptyChunks))

	// challenge: compute metadata hash
	hash := crypto.Keccak256Hash(blobBytes[0:62])
	copy(challengePreimage[0:], hash[:])

	// blob contains 131072 bytes but we can only utilize 31/32 of these
	if len(blobBytes) > 126976 {
		return nil, nil, fmt.Errorf("oversized batch payload")
	}

	// encode blob payload by prepending every 31 bytes with 1 zero byte
	index := 0

	for from := 0; from < len(blobBytes); from += 31 {
		to := from + 31
		if to > len(blobBytes) {
			to = len(blobBytes)
		}
		copy(blob[index+1:], blobBytes[from:to])
		index += 32
	}

	// compute z = challenge_digest % BLS_MODULUS
	challengeDigest := crypto.Keccak256Hash(challengePreimage[:])
	point := new(big.Int).Mod(new(big.Int).SetBytes(challengeDigest[:]), BLS_MODULUS)
	copy(z[:], point.Bytes()[0:32])

	return &blob, &z, nil
}

// NewDABatchFromBytes attempts to decode the given byte slice into a DABatch.
// Note: This function only populates the batch header, it leaves the blob-related fields empty.
func NewDABatchFromBytes(data []byte) (*DABatch, error) {
	if len(data) < 89 {
		return nil, fmt.Errorf("insufficient data for DABatch, expected at least 89 bytes but got %d", len(data))
	}

	b := &DABatch{
		Version:                data[0],
		BatchIndex:             binary.BigEndian.Uint64(data[1:9]),
		L1MessagePopped:        binary.BigEndian.Uint64(data[9:17]),
		TotalL1MessagePopped:   binary.BigEndian.Uint64(data[17:25]),
		DataHash:               common.BytesToHash(data[25:57]),
		BlobVersionedHash:      common.BytesToHash(data[57:89]),
		ParentBatchHash:        common.BytesToHash(data[89:121]),
		SkippedL1MessageBitmap: data[121:],
	}

	return b, nil
}

// Encode serializes the DABatch into bytes.
func (b *DABatch) Encode() []byte {
	batchBytes := make([]byte, 121+len(b.SkippedL1MessageBitmap))
	batchBytes[0] = b.Version
	binary.BigEndian.PutUint64(batchBytes[1:], b.BatchIndex)
	binary.BigEndian.PutUint64(batchBytes[9:], b.L1MessagePopped)
	binary.BigEndian.PutUint64(batchBytes[17:], b.TotalL1MessagePopped)
	copy(batchBytes[25:], b.DataHash[:])
	copy(batchBytes[57:], b.BlobVersionedHash[:])
	copy(batchBytes[89:], b.ParentBatchHash[:])
	copy(batchBytes[121:], b.SkippedL1MessageBitmap[:])
	return batchBytes
}

// Hash computes the hash of the serialized DABatch.
func (b *DABatch) Hash() common.Hash {
	bytes := b.Encode()
	return crypto.Keccak256Hash(bytes)
}

// DecodeFromCalldata attempts to decode a DABatch and an array of DAChunks from the provided calldata byte slice.
func DecodeFromCalldata(data []byte) (*DABatch, []*DAChunk, error) {
	// TODO: implement this function.
	return nil, nil, nil
}
