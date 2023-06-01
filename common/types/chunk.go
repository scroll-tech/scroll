package types

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// Chunk contains blocks to be encoded
type Chunk struct {
	Blocks []*WrappedBlock `json:"blocks"`
}

// Encode encodes the Chunk into RollupV2 Chunk Encoding.
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

	l2TxDataBytes := make([]byte, 0)

	for _, block := range c.Blocks {
		blockBytes, err := block.Encode()
		if err != nil {
			return nil, fmt.Errorf("failed to encode block: %v", err)
		}

		if len(blockBytes) != 60 {
			return nil, fmt.Errorf("block encoding is not 60 bytes long %x", len(blockBytes))
		}

		chunkBytes = append(chunkBytes, blockBytes...)

		// Append l2Tx Hashes
		for _, txData := range block.Transactions {
			if txData.Type == 0x7E {
				continue
			}
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
			l2TxDataBytes = append(l2TxDataBytes, txLen[:]...)
			l2TxDataBytes = append(l2TxDataBytes, rlpTxData...)
		}
	}

	chunkBytes = append(chunkBytes, l2TxDataBytes...)

	return chunkBytes, nil
}
