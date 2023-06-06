package store

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types/message"
)

func TestStack(t *testing.T) {
	// Create temp path
	path, err := os.MkdirTemp("/tmp/", "stack_db_test-")
	assert.NoError(t, err)
	defer os.RemoveAll(path)

	// Create stack db instance
	s, err := NewStack(filepath.Join(path, "test-stack"))
	assert.NoError(t, err)
	defer s.Close()

	for i := 0; i < 3; i++ {
		task := &ProvingTask{
			Task: &message.TaskMsg{
				ID: strconv.Itoa(i),
			},
			Times: 0,
		}

		err = s.Push(task)
		assert.NoError(t, err)
	}

	for i := 2; i >= 0; i-- {
		var peek *ProvingTask
		peek, err = s.Peek()
		assert.NoError(t, err)
		assert.Equal(t, strconv.Itoa(i), peek.Task.ID)
		err = s.Delete(strconv.Itoa(i))
		assert.NoError(t, err)
	}

	// test times
	task := &ProvingTask{
		Task: &message.TaskMsg{
			ID: strconv.Itoa(1),
		},
		Times: 0,
	}
	err = s.Push(task)
	assert.NoError(t, err)
	peek, err := s.Peek()
	assert.NoError(t, err)
	assert.Equal(t, 0, peek.Times)
	err = s.UpdateTimes(peek, 3)
	assert.NoError(t, err)

	peek2, err := s.Peek()
	assert.NoError(t, err)
	assert.Equal(t, 3, peek2.Times)
}
