package store

import (
	"encoding/binary"
	"encoding/json"
	"errors"

	"scroll-tech/common/message"

	"github.com/scroll-tech/go-ethereum/log"
	"go.etcd.io/bbolt"
)

var (
	// ErrEmpty empty error message
	ErrEmpty = errors.New("content is empty")
)

// Stack is a first-input last-output db.
type Stack struct {
	*bbolt.DB
}

// ProvingTraces is the value in stack.
// It contains traces and proved times.
type ProvingTraces struct {
	Traces *rollertypes.BlockTraces `json:"traces"`
	// Times is how many times roller proved.
	Times int `json:"times"`
}

var bucket = []byte("stack")

// NewStack new a Stack object.
func NewStack(path string) (*Stack, error) {
	kvdb, err := bbolt.Open(path, 0666, nil)
	if err != nil {
		return nil, err
	}
	err = kvdb.Update(func(tx *bbolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		log.Crit("init stack failed", "error", err)
	}
	return &Stack{DB: kvdb}, nil
}

// Push appends the block-traces on the top of Stack.
func (s *Stack) Push(traces *message.BlockTraces) error {
	byt, err := json.Marshal(traces)
	if err != nil {
		return err
	}
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, traces.Traces.ID)
	return s.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucket).Put(key, byt)
	})
}

// Pop pops the block-traces on the top of Stack.
func (s *Stack) Pop() (*message.BlockTraces, error) {
	var value []byte
	if err := s.Update(func(tx *bbolt.Tx) error {
		var key []byte
		bu := tx.Bucket(bucket)
		c := bu.Cursor()
		key, value = c.Last()
		return bu.Delete(key)
	}); err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, ErrEmpty
	}

	traces := &message.BlockTraces{}
	return traces, json.Unmarshal(value, traces)
}
