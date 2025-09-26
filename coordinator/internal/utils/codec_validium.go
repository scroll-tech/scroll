package utils

import (
	"encoding/binary"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
)

type CodecVersion uint8

const (
	daBatchValidiumEncodedLength = 137
)

type DABatch interface {
	SetHash(common.Hash)
}

type daBatchValidiumV1 struct {
	Version           CodecVersion `json:"version"`
	BatchIndex        uint64       `json:"batch_index"`
	BlobVersionedHash common.Hash  `json:"blob_versioned_hash"`
	ParentBatchHash   common.Hash  `json:"parent_batch_hash"`
	PostStateRoot     common.Hash  `json:"post_state_root"`
	WithDrawRoot      common.Hash  `json:"withdraw_root"`
	Commitment        common.Hash  `json:"commitment"`
}

type daBatchValidium struct {
	V1        *daBatchValidiumV1 `json:"V1,omitempty"`
	BatchHash common.Hash        `json:"batch_hash"`
}

func (da *daBatchValidium) SetHash(h common.Hash) {
	da.BatchHash = h
}

func FromVersion(v uint8) CodecVersion {
	return CodecVersion(v & STFVersionMask)
}

func (c CodecVersion) DABatchForTaskFromBytes(b []byte) (DABatch, error) {
	switch c {
	case 1:
		if v1, err := decodeDABatchV1(b); err == nil {
			return &daBatchValidium{
				V1: v1,
			}, nil
		} else {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown codec type %d", c)
	}
}

func decodeDABatchV1(data []byte) (*daBatchValidiumV1, error) {
	if len(data) != daBatchValidiumEncodedLength {
		return nil, fmt.Errorf("invalid data length for DABatchV7, expected %d bytes but got %d", daBatchValidiumEncodedLength, len(data))
	}

	const (
		versionSize = 1
		indexSize   = 8
		hashSize    = 32
	)

	// Offsets (same as encodeBatchHeaderValidium)
	versionOffset := 0
	indexOffset := versionOffset + versionSize
	parentHashOffset := indexOffset + indexSize
	stateRootOffset := parentHashOffset + hashSize
	withdrawRootOffset := stateRootOffset + hashSize
	commitmentOffset := withdrawRootOffset + hashSize

	version := CodecVersion(data[versionOffset])
	batchIndex := binary.BigEndian.Uint64(data[indexOffset : indexOffset+indexSize])
	parentBatchHash := common.BytesToHash(data[parentHashOffset : parentHashOffset+hashSize])
	postStateRoot := common.BytesToHash(data[stateRootOffset : stateRootOffset+hashSize])
	withdrawRoot := common.BytesToHash(data[withdrawRootOffset : withdrawRootOffset+hashSize])
	commitment := common.BytesToHash(data[commitmentOffset : commitmentOffset+hashSize])

	return &daBatchValidiumV1{
		Version:         version,
		BatchIndex:      batchIndex,
		ParentBatchHash: parentBatchHash,
		PostStateRoot:   postStateRoot,
		WithDrawRoot:    withdrawRoot,
		Commitment:      commitment,
	}, nil
}
