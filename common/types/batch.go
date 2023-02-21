package types

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"

	abi "scroll-tech/bridge/abi"
)

// PublicInputHashConfig is the input config of batch hash.
type PublicInputHashConfig struct {
	MaxTxNum      int
	PaddingTxHash common.Hash
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
	hash  *common.Hash
	piCfg *PublicInputHashConfig
}

// Hash calculates hash of batches.
func (b *BatchData) Hash() *common.Hash {
	if b.hash != nil {
		return b.hash
	}

	buf := make([]byte, 8)
	hasher := crypto.NewKeccakState()

	// 1. hash PrevStateRoot, NewStateRoot, WithdrawTrieRoot
	hasher.Write(b.Batch.PrevStateRoot[:])
	hasher.Write(b.Batch.NewStateRoot[:])
	hasher.Write(b.Batch.WithdrawTrieRoot[:])

	// 2. hash all block contexts
	for _, block := range b.Batch.Blocks {
		// write BlockHash & ParentHash
		hasher.Write(block.BlockHash[:])
		hasher.Write(block.ParentHash[:])
		// write BlockNumber
		binary.BigEndian.PutUint64(buf, block.BlockNumber)
		hasher.Write(buf)
		// write Timestamp
		binary.BigEndian.PutUint64(buf, block.Timestamp)
		hasher.Write(buf)
		// write BaseFee
		var baseFee [32]byte
		if block.BaseFee != nil {
			baseFee = newByte32FromBytes(block.BaseFee.Bytes())
		}
		hasher.Write(baseFee[:])
		// write GasLimit
		binary.BigEndian.PutUint64(buf, block.GasLimit)
		hasher.Write(buf)
		// write NumTransactions
		binary.BigEndian.PutUint16(buf[:2], block.NumTransactions)
		hasher.Write(buf[:2])
		// write NumL1Messages
		binary.BigEndian.PutUint16(buf[:2], block.NumL1Messages)
		hasher.Write(buf[:2])
	}

	// 3. add all tx hashes
	for _, txHash := range b.TxHashes {
		hasher.Write(txHash[:])
	}

	// 4. append empty tx hash up to MaxTxNum
	maxTxNum := defaultMaxTxNum
	paddingTxHash := common.Hash(defaultPaddingTxHash)
	if b.piCfg != nil {
		maxTxNum = b.piCfg.MaxTxNum
		paddingTxHash = b.piCfg.PaddingTxHash
	}
	for i := len(b.TxHashes); i < maxTxNum; i++ {
		hasher.Write(paddingTxHash[:])
	}

	b.hash = new(common.Hash)
	hasher.Read(b.hash[:])

	return b.hash
}

// NewBatchData generates batches to committed based on parentBatch and blockTraces.
func NewBatchData(parentBatch *BlockBatch, blockTraces []*types.BlockTrace) *BatchData {
	batchData := new(BatchData)
	batch := &batchData.Batch

	// set BatchIndex, ParentBatchHash
	batch.BatchIndex = parentBatch.Index + 1
	batch.ParentBatchHash = common.HexToHash(parentBatch.Hash)
	batch.Blocks = make([]abi.IScrollChainBlockContext, len(blockTraces))

	var batchTxDataBuf bytes.Buffer
	batchTxDataWriter := bufio.NewWriter(&batchTxDataBuf)

	for i, trace := range blockTraces {
		batchData.TotalTxNum += uint64(len(trace.Transactions))
		batchData.TotalL2Gas += trace.Header.GasUsed

		// set baseFee to 0 when it's nil in the block header
		baseFee := trace.Header.BaseFee
		if baseFee == nil {
			baseFee = big.NewInt(0)
		}

		batch.Blocks[i] = abi.IScrollChainBlockContext{
			BlockHash:       trace.Header.Hash(),
			ParentHash:      trace.Header.ParentHash,
			BlockNumber:     trace.Header.Number.Uint64(),
			Timestamp:       trace.Header.Time,
			BaseFee:         baseFee,
			GasLimit:        trace.Header.GasLimit,
			NumTransactions: uint16(len(trace.Transactions)),
			NumL1Messages:   0, // TODO: currently use 0, will re-enable after we use l2geth to include L1 messages
		}

		// fill in RLP-encoded transactions
		for _, txData := range trace.Transactions {
			// right now we only support legacy tx
			tx := types.NewTx(&types.LegacyTx{
				Nonce:    txData.Nonce,
				To:       txData.To,
				Value:    txData.Value.ToInt(),
				Gas:      txData.Gas,
				GasPrice: txData.GasPrice.ToInt(),
				Data:     []byte(txData.Data),
				V:        txData.V.ToInt(),
				R:        txData.R.ToInt(),
				S:        txData.S.ToInt(),
			})
			var rlpBuf bytes.Buffer
			writer := bufio.NewWriter(&rlpBuf)
			_ = tx.EncodeRLP(writer)
			rlpTxData := rlpBuf.Bytes()
			var txLen [4]byte
			binary.BigEndian.PutUint32(txLen[:], uint32(len(rlpTxData)))
			batchTxDataWriter.Write(txLen[:])
			batchTxDataWriter.Write(rlpTxData)
			batchData.TxHashes = append(batchData.TxHashes, tx.Hash())
		}

		// set PrevStateRoot from the first block
		if i == 0 {
			batch.PrevStateRoot = trace.StorageTrace.RootBefore
		}

		// set NewStateRoot & WithdrawTrieRoot from the last block
		if i == len(blockTraces)-1 {
			batch.NewStateRoot = trace.Header.Root
			batch.WithdrawTrieRoot = trace.WithdrawTrieRoot
		}
	}

	batch.L2Transactions = batchTxDataBuf.Bytes()

	return batchData
}

// NewGenesisBatchData generates batches to committed based on parentBatch and blockTraces.
func NewGenesisBatchData(blockTraces []*types.BlockTrace) *BatchData {
	batchData := new(BatchData)
	batch := &batchData.Batch

	// set BatchIndex, ParentBatchHash
	batch.BatchIndex = 1
	batch.Blocks = make([]abi.IScrollChainBlockContext, len(blockTraces))

	var batchTxDataBuf bytes.Buffer
	batchTxDataWriter := bufio.NewWriter(&batchTxDataBuf)

	for i, trace := range blockTraces {
		batchData.TotalTxNum += uint64(len(trace.Transactions))
		batchData.TotalL2Gas += trace.Header.GasUsed

		batch.Blocks[i] = abi.IScrollChainBlockContext{
			BlockHash:       trace.Header.Hash(),
			ParentHash:      trace.Header.ParentHash,
			BlockNumber:     trace.Header.Number.Uint64(),
			Timestamp:       trace.Header.Time,
			BaseFee:         trace.Header.BaseFee,
			GasLimit:        trace.Header.GasLimit,
			NumTransactions: uint16(len(trace.Transactions)),
			NumL1Messages:   0, // TODO: currently use 0, will re-enable after we use l2geth to include L1 messages
		}

		// fill in RLP-encoded transactions
		for _, txData := range trace.Transactions {
			// right now we only support legacy tx
			tx := types.NewTx(&types.LegacyTx{
				Nonce:    txData.Nonce,
				To:       txData.To,
				Value:    txData.Value.ToInt(),
				Gas:      txData.Gas,
				GasPrice: txData.GasPrice.ToInt(),
				Data:     []byte(txData.Data),
				V:        txData.V.ToInt(),
				R:        txData.R.ToInt(),
				S:        txData.S.ToInt(),
			})
			var rlpBuf bytes.Buffer
			writer := bufio.NewWriter(&rlpBuf)
			_ = tx.EncodeRLP(writer)
			rlpTxData := rlpBuf.Bytes()
			var txLen [4]byte
			binary.BigEndian.PutUint32(txLen[:], uint32(len(rlpTxData)))
			batchTxDataWriter.Write(txLen[:])
			batchTxDataWriter.Write(rlpTxData)
			batchData.TxHashes = append(batchData.TxHashes, tx.Hash())
		}

		// set NewStateRoot & WithdrawTrieRoot from the last block
		if i == len(blockTraces)-1 {
			batch.NewStateRoot = trace.Header.Root
			batch.WithdrawTrieRoot = trace.WithdrawTrieRoot
		}
	}

	batch.L2Transactions = batchTxDataBuf.Bytes()

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
