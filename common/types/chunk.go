package types

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
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

	var chunkBytes []byte
	chunkBytes = append(chunkBytes, byte(numBlocks))

	var l2TxDataBytes []byte

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

// Hash hashes the Chunk into RollupV2 Chunk Hash
func (c *Chunk) Hash() ([]byte, error) {
	chunkBytes, err := c.Encode()
	if err != nil {
		return nil, err
	}
	numBlocks := chunkBytes[0]

	// concatenate block contexts
	// only first 58 bytes is needed
	dataBytes := make([]byte, 0)
	for i := 0; i < int(numBlocks); i++ {
		// only first 58 bytes is needed
		dataBytes = append(dataBytes, chunkBytes[1+60*i:60*i+59]...)
	}

	// concatenate l1 and l2 tx hashes
	l2TxHashes := make([]byte, 0)
	for _, block := range c.Blocks {
		for _, txData := range block.Transactions {
			// TODO: concatenate l1 message hashes
			if txData.Type == 0x7E {
				continue
			}
			// concatenate l2 txs hashes
			// retrieve the number of transactions in current block.
			txHash := strings.TrimPrefix(txData.TxHash, "0x")
			hashBytes, err := hex.DecodeString(txHash)
			if err != nil {
				return nil, err
			}
			l2TxHashes = append(l2TxHashes, hashBytes...)
		}
	}

	dataBytes = append(dataBytes, l2TxHashes...)
	hash := crypto.Keccak256Hash(dataBytes).Bytes()
	return hash, nil
}
