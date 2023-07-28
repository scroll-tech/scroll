package store

import (
	"encoding/json"
	"errors"

	"github.com/scroll-tech/go-ethereum/log"
	"go.etcd.io/bbolt"

	"scroll-tech/common/types/message"
)

var (
	// ErrEmpty empty error message
	ErrEmpty = errors.New("content is empty")
)

// Stack is a first-input last-output db.
type Stack struct {
	*bbolt.DB
}

// ProvingTask is the value in stack.
// It contains TaskMsg and proved times.
type ProvingTask struct {
	Task *message.TaskMsg `json:"task"`
	// Times is how many times prover proved.
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

// Push appends the proving-task on the top of Stack.
func (s *Stack) Push(task *ProvingTask) error {
	byt, err := json.Marshal(task)
	if err != nil {
		return err
	}
	key := []byte(task.Task.ID)
	return s.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucket).Put(key, byt)
	})
}

// Peek return the top element of the Stack.
func (s *Stack) Peek() (*ProvingTask, error) {
	var value []byte
	if err := s.View(func(tx *bbolt.Tx) error {
		bu := tx.Bucket(bucket)
		c := bu.Cursor()
		_, value = c.Last()
		return nil
	}); err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, ErrEmpty
	}

	traces := &ProvingTask{}
	err := json.Unmarshal(value, traces)
	if err != nil {
		return nil, err
	}
	return traces, nil
}

// Delete pops the proving-task on the top of Stack.
func (s *Stack) Delete(taskID string) error {
	return s.Update(func(tx *bbolt.Tx) error {
		bu := tx.Bucket(bucket)
		return bu.Delete([]byte(taskID))
	})
}

// UpdateTimes udpates the prover prove times of the proving task.
func (s *Stack) UpdateTimes(task *ProvingTask, udpateTimes int) error {
	task.Times = udpateTimes
	byt, err := json.Marshal(task)
	if err != nil {
		return err
	}
	key := []byte(task.Task.ID)
	return s.Update(func(tx *bbolt.Tx) error {
		bu := tx.Bucket(bucket)
		c := bu.Cursor()
		key, _ = c.Last()
		return bu.Put(key, byt)
	})
}
