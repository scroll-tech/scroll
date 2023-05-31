package types

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
)

type Chunk struct {
	Blocks []*WrappedBlock `json:"blocks"`
}

// encode chunk
func (c *Chunk) Encode() ([]byte, error) {
	numBlocks := len(c.Blocks)

	if numBlocks > 255 {
		return nil, errors.New("number of blocks exceeds 1 byte")
	}
	if numBlocks == 0 {
		return nil, errors.New("number of blocks is 0")
	}

	chunkBytes := make([]byte, 0)
	chunkBytes = append(chunkBytes, byte(numBlocks))

	var batchTxDataBuf bytes.Buffer
	batchTxDataWriter := bufio.NewWriter(&batchTxDataBuf)

	for _, block := range c.Blocks {
		blockBytes, err := block.Encode()
		if err != nil {
			return nil, fmt.Errorf("failed to encode block: %v", err)
		}

		if len(blockBytes) != 60 {
			return nil, fmt.Errorf("block encoding is not 60 bytes long %x", len(blockBytes))
		}

		chunkBytes = append(chunkBytes, blockBytes...)

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
		}
	}

	if err := batchTxDataWriter.Flush(); err != nil {
		panic("Buffered I/O flush failed")
	}

	chunkBytes = append(chunkBytes, batchTxDataBuf.Bytes()...)

	return chunkBytes, nil
}

// calculate chunk data hash
func (c *Chunk) Hash() ([]byte, error) {
	chunkCodec, err := c.Encode()

	if err != nil {
		return nil, err
	}

	numBlocks := chunkCodec[0]

	// concatenate block contexts
	dataBytes := chunkCodec[1 : 60*numBlocks+1]

	// retrieve block context start index
	blockIndex := 1

	// retrieve l2 tx start index
	l2TxIndex := uint32(60*numBlocks + 1)

	// l2 tx hashes
	l2TxHashes := make([]byte, 0)

	for numBlocks > 0 {
		// TODO: concatenate l1 message hashes

		// concatenate l2 txs hashes
		// retrieve the number of transactions in current block.
		numTransactionsIndex := blockIndex + 56
		numTxsInBlock := binary.BigEndian.Uint16(chunkCodec[numTransactionsIndex : numTransactionsIndex+2])

		for numTxsInBlock > 0 {
			l2TxLen := binary.BigEndian.Uint32(chunkCodec[l2TxIndex : l2TxIndex+4])
			l2TxIndex += 4
			txPayload := chunkCodec[l2TxIndex : l2TxIndex+l2TxLen]
			txHash := crypto.Keccak256Hash(txPayload).Bytes()
			l2TxIndex += l2TxLen

			l2TxHashes = append(l2TxHashes, txHash...)
			numTxsInBlock--
		}

		numBlocks--
		blockIndex += 60
	}
	dataBytes = append(dataBytes, l2TxHashes...)

	// TODO: check the number of L2 transactions in the chunk

	// TODO: check chunk has correct length

	// hash data bytes
	hash := crypto.Keccak256Hash(dataBytes).Bytes()

	return hash, nil
}
