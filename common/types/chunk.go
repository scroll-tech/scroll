package types

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
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

// decode chunk
func DecodeChunk([]byte) (*Chunk, error) {
	return nil, nil
}

// calculate chunk data hash
func (c *Chunk) Hash() *common.Hash {
	return nil
}