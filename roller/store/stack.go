package store

import (
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/scroll-tech/go-ethereum/log"
	"go.etcd.io/bbolt"

	rollertypes "scroll-tech/common/message"
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
func (s *Stack) Push(traces *ProvingTraces) error {
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

// Peak return the top element of the stack
func (s *Stack) Peak() (*ProvingTraces, error) {
	var value []byte
	if err := s.View(func(tx *bbolt.Tx) error {
		bu := tx.Bucket(bucket)
		c := bu.Cursor()
		_, value = c.Last()
		if len(value) == 0 {
			return ErrEmpty
		}
		return nil
	}); err != nil {
		return nil, err
	}

	traces := &ProvingTraces{}
	err := json.Unmarshal(value, traces)
	if err != nil {
		return nil, err
	}
	// notice return pointer of the trace
	return traces, nil
}

// Pop pops the block-traces on the top of Stack.
func (s *Stack) Pop() (*ProvingTraces, error) {
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

	traces := &ProvingTraces{}
	err := json.Unmarshal(value, traces)
	if err != nil {
		return nil, err
	}
	return traces, nil
}
