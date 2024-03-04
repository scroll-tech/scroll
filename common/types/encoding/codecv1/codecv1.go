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

	"scroll-tech/common/types/encoding"
)

const CodecV1Version = 1

type DABlock struct {
	BlockNumber     uint64
	Timestamp       uint64
	BaseFee         *big.Int
	GasLimit        uint64
	NumTransactions uint16
	NumL1Messages   uint16
}

type DAChunk struct {
	Blocks       []*DABlock
	Transactions [][]*types.TransactionData
}

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
	sidecar *types.BlobTxSidecar
}

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

func (b *DABlock) Encode() ([]byte, error) {
	bytes := make([]byte, 60)
	binary.BigEndian.PutUint64(bytes[0:], b.BlockNumber)
	binary.BigEndian.PutUint64(bytes[8:], b.Timestamp)
	// TODO: [16:47] Currently, baseFee is 0, because we disable EIP-1559.
	binary.BigEndian.PutUint64(bytes[48:], b.GasLimit)
	binary.BigEndian.PutUint16(bytes[56:], b.NumTransactions)
	binary.BigEndian.PutUint16(bytes[58:], b.NumL1Messages)
	return bytes, nil
}

func NewDAChunk(chunk *encoding.Chunk, totalL1MessagePoppedBefore uint64) (*DAChunk, error) {
	var blocks []*DABlock
	var txs [][]*types.TransactionData

	for _, block := range chunk.Blocks {
		b, _ := NewDABlock(block, totalL1MessagePoppedBefore)
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

func (c *DAChunk) Encode() ([]byte, error) {
	var chunkBytes []byte
	chunkBytes = append(chunkBytes, byte(len(c.Blocks)))

	for _, block := range c.Blocks {
		blockBytes, _ := block.Encode()
		chunkBytes = append(chunkBytes, blockBytes...)
	}

	return chunkBytes, nil
}

func (c *DAChunk) Hash() (common.Hash, error) {
	chunkBytes, err := c.Encode()
	if err != nil {
		return common.Hash{}, err
	}
	numBlocks := chunkBytes[0]

	// concatenate block contexts
	var dataBytes []byte
	for i := 0; i < int(numBlocks); i++ {
		// only the first 58 bytes of each BlockContext are needed for the hashing process
		dataBytes = append(dataBytes, chunkBytes[1+60*i:60*i+59]...)
	}

	// concatenate l1 tx hashes
	for _, blockTxs := range c.Transactions {
		for _, txData := range blockTxs {
			txHash := strings.TrimPrefix(txData.TxHash, "0x")
			hashBytes, err := hex.DecodeString(txHash)
			if err != nil {
				return common.Hash{}, err
			}
			if txData.Type == types.L1MessageTxType {
				dataBytes = append(dataBytes, hashBytes...)
			}
		}
	}

	hash := crypto.Keccak256Hash(dataBytes)
	return hash, nil
}

func NewDABatch(batch *encoding.Batch, totalL1MessagePoppedBefore uint64) (*DABatch, error) {
	// buffer for storing chunk hashes in order to compute the batch data hash
	var dataBytes []byte

	// skipped L1 message bitmap, an array of 256-bit bitmaps
	var skippedBitmap []*big.Int

	// the first queue index that belongs to this batch
	baseIndex := batch.TotalL1MessagePoppedBefore

	// the next queue index that we need to process
	nextIndex := batch.TotalL1MessagePoppedBefore

	blobPayload := make([]byte, 31)

	// this encoding can only support up to 15 chunks per batch
	if len(batch.Chunks) > 15 {
		return nil, fmt.Errorf("too many chunks in batch")
	}

	// encode metadata
	blobPayload[0] = byte(len(batch.Chunks))

	for chunkID, chunk := range batch.Chunks {
		// encode metadata
		size := uint16(len(chunk.Blocks))
		binary.BigEndian.PutUint16(blobPayload[1+2*chunkID:], size)

		// build data hash
		totalL1MessagePoppedBeforeChunk := nextIndex
		daChunk, _ := NewDAChunk(chunk, totalL1MessagePoppedBeforeChunk)
		chunkHash, err := daChunk.Hash()
		if err != nil {
			return nil, err
		}
		dataBytes = append(dataBytes, chunkHash.Bytes()...)

		// build skip bitmap
		for blockID, block := range chunk.Blocks {
			for _, tx := range block.Transactions {
				// encode L2 txs into blob payload
				if tx.Type != types.L1MessageTxType {
					rlpTxData, err := encoding.ConvertTxDataToRLPEncoding(tx)
					if err != nil {
						return nil, err
					}
					blobPayload = append(blobPayload, rlpTxData...)
					continue
				}

				currentIndex := tx.Nonce

				if currentIndex < nextIndex {
					return nil, fmt.Errorf("unexpected batch payload, expected queue index: %d, got: %d. Batch index: %d, chunk index in batch: %d, block index in chunk: %d, block hash: %v, transaction hash: %v", nextIndex, currentIndex, batch.Index, chunkID, blockID, block.Header.Hash(), tx.TxHash)
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

	// compute data hash
	dataHash := crypto.Keccak256Hash(dataBytes)

	// compute skipped bitmap
	bitmapBytes := make([]byte, len(skippedBitmap)*32)
	for ii, num := range skippedBitmap {
		bytes := num.Bytes()
		padding := 32 - len(bytes)
		copy(bitmapBytes[32*ii+padding:], bytes)
	}

	// blob contains 131072 bytes but we can only utilize 31/32 of these
	if len(blobPayload) > 126976 {
		return nil, fmt.Errorf("oversized batch payload")
	}

	// encode into blob by prepending every 31 bytes with 1 zero byte
	var blob kzg4844.Blob
	index := 0

	for from := 0; from < len(blobPayload); from += 31 {
		to := from + 31
		if to > len(blobPayload) {
			to = len(blobPayload)
		}
		copy(blob[index+1:], blobPayload[from:to])
		index += 32
	}

	// create sidecar
	c, err := kzg4844.BlobToCommitment(blob)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob commitment")
	}
	p, _ := kzg4844.ComputeBlobProof(blob, c)
	if err != nil {
		return nil, fmt.Errorf("failed to compute blob proof")
	}

	sidecar := &types.BlobTxSidecar{
		Blobs:       []kzg4844.Blob{blob},
		Commitments: []kzg4844.Commitment{c},
		Proofs:      []kzg4844.Proof{p},
	}

	hasher := sha256.New()
	blobVersionedHash := kzg4844.CalcBlobHashV1(hasher, &c)

	daBatch := DABatch{
		Version:                CodecV1Version,
		BatchIndex:             batch.Index,
		L1MessagePopped:        nextIndex - totalL1MessagePoppedBefore,
		TotalL1MessagePopped:   nextIndex,
		DataHash:               dataHash,
		BlobVersionedHash:      blobVersionedHash,
		ParentBatchHash:        batch.ParentBatchHash,
		SkippedL1MessageBitmap: bitmapBytes,
		sidecar:                sidecar,
	}

	return &daBatch, nil
}

func (b *DABatch) Encode() ([]byte, error) {
	batchBytes := make([]byte, 121+len(b.SkippedL1MessageBitmap))
	batchBytes[0] = b.Version
	binary.BigEndian.PutUint64(batchBytes[1:], b.BatchIndex)
	binary.BigEndian.PutUint64(batchBytes[9:], b.L1MessagePopped)
	binary.BigEndian.PutUint64(batchBytes[17:], b.TotalL1MessagePopped)
	copy(batchBytes[25:], b.DataHash[:])
	copy(batchBytes[57:], b.BlobVersionedHash[:])
	copy(batchBytes[89:], b.ParentBatchHash[:])
	copy(batchBytes[121:], b.SkippedL1MessageBitmap[:])
	return batchBytes, nil
}

func (b *DABatch) Hash() (common.Hash, error) {
	bytes, _ := b.Encode()
	return crypto.Keccak256Hash(bytes), nil
}

func DecodeFromCalldata(data []byte) (*DABatch, []*DAChunk, error) {
	return nil, nil, nil
}
