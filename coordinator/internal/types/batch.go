package types

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"

	abi "scroll-tech/bridge/abi"
)

// PublicInputHashConfig is the configuration of how to compute the public input hash.
type PublicInputHashConfig struct {
	MaxTxNum      int         `json:"max_tx_num"`
	PaddingTxHash common.Hash `json:"padding_tx_hash"`
}

const defaultMaxTxNum = 44

var defaultPaddingTxHash = [32]byte{}

// BatchData contains info of batch to be committed.
type BatchData struct {
	Batch        abi.IScrollChainBatch
	TxHashes     []common.Hash
	TotalTxNum   uint64
	TotalL1TxNum uint64
	TotalL2Gas   uint64

	// cache for the BatchHash
	hash *common.Hash
	// The config to compute the public input hash, or the block hash.
	// If it is nil, the hash calculation will use `defaultMaxTxNum` and `defaultPaddingTxHash`.
	piCfg *PublicInputHashConfig
}

// Timestamp returns the timestamp of the first block in the BlockData.
func (b *BatchData) Timestamp() uint64 {
	if len(b.Batch.Blocks) == 0 {
		return 0
	}
	return b.Batch.Blocks[0].Timestamp
}

// Hash calculates the hash of this batch.
func (b *BatchData) Hash() *common.Hash {
	if b.hash != nil {
		return b.hash
	}

	buf := make([]byte, 8)
	hasher := crypto.NewKeccakState()

	// 1. hash PrevStateRoot, NewStateRoot, WithdrawTrieRoot
	// @todo: panic on error here.
	_, _ = hasher.Write(b.Batch.PrevStateRoot[:])
	_, _ = hasher.Write(b.Batch.NewStateRoot[:])
	_, _ = hasher.Write(b.Batch.WithdrawTrieRoot[:])

	// 2. hash all block contexts
	for _, block := range b.Batch.Blocks {
		// write BlockHash & ParentHash
		_, _ = hasher.Write(block.BlockHash[:])
		_, _ = hasher.Write(block.ParentHash[:])
		// write BlockNumber
		binary.BigEndian.PutUint64(buf, block.BlockNumber)
		_, _ = hasher.Write(buf)
		// write Timestamp
		binary.BigEndian.PutUint64(buf, block.Timestamp)
		_, _ = hasher.Write(buf)
		// write BaseFee
		var baseFee [32]byte
		if block.BaseFee != nil {
			baseFee = newByte32FromBytes(block.BaseFee.Bytes())
		}
		_, _ = hasher.Write(baseFee[:])
		// write GasLimit
		binary.BigEndian.PutUint64(buf, block.GasLimit)
		_, _ = hasher.Write(buf)
		// write NumTransactions
		binary.BigEndian.PutUint16(buf[:2], block.NumTransactions)
		_, _ = hasher.Write(buf[:2])
		// write NumL1Messages
		binary.BigEndian.PutUint16(buf[:2], block.NumL1Messages)
		_, _ = hasher.Write(buf[:2])
	}

	// 3. add all tx hashes
	for _, txHash := range b.TxHashes {
		_, _ = hasher.Write(txHash[:])
	}

	// 4. append empty tx hash up to MaxTxNum
	maxTxNum := defaultMaxTxNum
	paddingTxHash := common.Hash(defaultPaddingTxHash)
	if b.piCfg != nil {
		maxTxNum = b.piCfg.MaxTxNum
		paddingTxHash = b.piCfg.PaddingTxHash
	}
	for i := len(b.TxHashes); i < maxTxNum; i++ {
		_, _ = hasher.Write(paddingTxHash[:])
	}

	b.hash = new(common.Hash)
	_, _ = hasher.Read(b.hash[:])

	return b.hash
}

// NewBatchData creates a BatchData given the parent batch information and the traces of the blocks
// included in this batch
func NewBatchData(parentBatch *BatchInfo, blocks []*WrappedBlock, piCfg *PublicInputHashConfig) *BatchData {
	batchData := new(BatchData)
	batch := &batchData.Batch

	// set BatchIndex, ParentBatchHash
	batch.BatchIndex = parentBatch.Index + 1
	batch.ParentBatchHash = common.HexToHash(parentBatch.Hash)
	batch.Blocks = make([]abi.IScrollChainBlockContext, len(blocks))

	var batchTxDataBuf bytes.Buffer
	batchTxDataWriter := bufio.NewWriter(&batchTxDataBuf)

	for i, block := range blocks {
		batchData.TotalTxNum += uint64(len(block.Transactions))
		batchData.TotalL2Gas += block.Header.GasUsed

		// set baseFee to 0 when it's nil in the block header
		baseFee := block.Header.BaseFee
		if baseFee == nil {
			baseFee = big.NewInt(0)
		}

		batch.Blocks[i] = abi.IScrollChainBlockContext{
			BlockHash:       block.Header.Hash(),
			ParentHash:      block.Header.ParentHash,
			BlockNumber:     block.Header.Number.Uint64(),
			Timestamp:       block.Header.Time,
			BaseFee:         baseFee,
			GasLimit:        block.Header.GasLimit,
			NumTransactions: uint16(len(block.Transactions)),
			NumL1Messages:   0, // TODO: currently use 0, will re-enable after we use l2geth to include L1 messages
		}

		// fill in RLP-encoded transactions
		for _, txData := range block.Transactions {
			data, _ := hexutil.Decode(txData.Data)
			// right now we only support legacy tx
			tx := types.NewTx(&types.LegacyTx{
				Nonce:    txData.Nonce,
				To:       txData.To,
				Value:    txData.Value.ToInt(),
				Gas:      txData.Gas,
				GasPrice: txData.GasPrice.ToInt(),
				Data:     data,
				V:        txData.V.ToInt(),
				R:        txData.R.ToInt(),
				S:        txData.S.ToInt(),
			})
			rlpTxData, _ := tx.MarshalBinary()
			var txLen [4]byte
			binary.BigEndian.PutUint32(txLen[:], uint32(len(rlpTxData)))
			_, _ = batchTxDataWriter.Write(txLen[:])
			_, _ = batchTxDataWriter.Write(rlpTxData)
			batchData.TxHashes = append(batchData.TxHashes, tx.Hash())
		}

		if i == 0 {
			batch.PrevStateRoot = common.HexToHash(parentBatch.StateRoot)
		}

		// set NewStateRoot & WithdrawTrieRoot from the last block
		if i == len(blocks)-1 {
			batch.NewStateRoot = block.Header.Root
			batch.WithdrawTrieRoot = block.WithdrawTrieRoot
		}
	}

	if err := batchTxDataWriter.Flush(); err != nil {
		panic("Buffered I/O flush failed")
	}

	batch.L2Transactions = batchTxDataBuf.Bytes()
	batchData.piCfg = piCfg

	return batchData
}

// NewGenesisBatchData generates the batch that contains the genesis block.
func NewGenesisBatchData(genesisBlockTrace *WrappedBlock) *BatchData {
	header := genesisBlockTrace.Header
	if header.Number.Uint64() != 0 {
		panic("invalid genesis block trace: block number is not 0")
	}

	batchData := new(BatchData)
	batch := &batchData.Batch

	// fill in batch information
	batch.BatchIndex = 0
	batch.Blocks = make([]abi.IScrollChainBlockContext, 1)
	batch.NewStateRoot = header.Root
	// PrevStateRoot, WithdrawTrieRoot, ParentBatchHash should all be 0
	// L2Transactions should be empty

	// fill in block context
	batch.Blocks[0] = abi.IScrollChainBlockContext{
		BlockHash:       header.Hash(),
		ParentHash:      header.ParentHash,
		BlockNumber:     header.Number.Uint64(),
		Timestamp:       header.Time,
		BaseFee:         header.BaseFee,
		GasLimit:        header.GasLimit,
		NumTransactions: 0,
		NumL1Messages:   0,
	}

	return batchData
}

// newByte32FromBytes converts the bytes in big-endian encoding to 32 bytes in big-endian encoding
func newByte32FromBytes(b []byte) [32]byte {
	var byte32 [32]byte

	if len(b) > 32 {
		b = b[len(b)-32:]
	}

	copy(byte32[32-len(b):], b)
	return byte32
}
