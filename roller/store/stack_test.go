package store

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/message"
)

func TestStack(t *testing.T) {
	// Create temp path
	path, err := ioutil.TempDir("/tmp/", "stack_db_test-")
	assert.NoError(t, err)
	defer os.RemoveAll(path)

	// Create stack db instance
	s, err := NewStack(filepath.Join(path, "test-stack"))
	assert.NoError(t, err)
	defer s.Close()

	for i := 0; i < 3; i++ {
		task := &ProvingTask{
			Task: &message.TaskMsg{
				ID:     strconv.Itoa(i),
				Traces: nil,
			},
			Times: 0,
		}

		err = s.Push(task)
		assert.NoError(t, err)
	}

	for i := 2; i >= 0; i-- {
		var pop *ProvingTask
		pop, err = s.Pop()
		assert.NoError(t, err)
		assert.Equal(t, strconv.Itoa(i), pop.Task.ID)
	}

	// test times
	task := &ProvingTask{
		Task: &message.TaskMsg{
			ID:     strconv.Itoa(1),
			Traces: nil,
		},
		Times: 0,
	}
	err = s.Push(task)
	assert.NoError(t, err)
	pop, err := s.Pop()
	assert.NoError(t, err)
	err = s.Push(pop)
	assert.NoError(t, err)

	pop2, err := s.Pop()
	assert.NoError(t, err)
	assert.Equal(t, 2, pop2.Times)
}
