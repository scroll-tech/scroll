package types

import "github.com/scroll-tech/go-ethereum/common"

type Batch struct {
	Chunks []*Chunk
	// ... other fields as needed
}

func NewBatch(version uint8, batchIndex, totalL1MessagePoppedBefore uint64, parentBatchHash common.Hash, chunks []*Chunk) *Batch {
	// calculate `l1MessagePopped`, `totalL1MessagePopped`, `dataHash`, and `skippedL1MessageBitmap` based on `chunks`
	batch := &Batch{
		Chunks: chunks,
		// ... initialize other fields as needed
	}
	return batch
}

// encode batch
// reference: https://github.com/scroll-tech/scroll/blob/develop/contracts/src/libraries/codec/BatchHeaderV0Codec.sol#L5
func (b *Batch) Encode() []byte {
	// encode `b.Chunks` along with other fields
	return nil
}

// calculate batch hash
// reference: https://github.com/scroll-tech/scroll/blob/develop/contracts/src/L1/rollup/ScrollChain.sol#L394
func (b *Batch) Hash() *common.Hash {
	// calculate hash based on `b.Chunks` and other fields
	return nil
}
