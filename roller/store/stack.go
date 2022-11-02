package store

import (
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/scroll-tech/go-ethereum/log"
	"go.etcd.io/bbolt"

	"scroll-tech/common/message"
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

// Push appends the proving-task on the top of Stack.
func (s *Stack) Push(task *ProvingTask) error {
	byt, err := json.Marshal(task)
	if err != nil {
		return err
	}
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, task.Task.ID)
	return s.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucket).Put(key, byt)
	})
}

// Pop pops the proving-task on the top of Stack.
func (s *Stack) Pop() (*ProvingTask, error) {
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

	task := &ProvingTask{}
	err := json.Unmarshal(value, task)
	if err != nil {
		return nil, err
	}
	task.Times++
	return task, nil
}
