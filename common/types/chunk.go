package types

import (
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
)

type Chunk struct {
	Blocks []*WrappedBlock
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

	bytes := make([]byte, 0)
	bytes = append(bytes, byte(numBlocks))

	for _, block := range c.Blocks {
		blockBytes, err := block.Encode()
		if err != nil {
				return nil, fmt.Errorf("failed to encode block: %v", err)
		}

		if len(blockBytes) != 60 {
				return nil, fmt.Errorf("block encoding is not 60 bytes long %x", len(blockBytes))
		}

		bytes = append(bytes, blockBytes...)
	}

	// TODO: add raw rlp encoded L2 Txs

	return bytes, nil
}

// decode chunk
func DecodeChunk([]byte) (*Chunk, error) {
	return nil, nil
}

// calculate chunk data hash
func (c *Chunk) Hash() *common.Hash {
	return nil
}