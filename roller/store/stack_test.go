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
		task := &message.TaskMsg{
			ID:     strconv.Itoa(i),
			Traces: nil,
		}
		err = s.Push(task)
		assert.NoError(t, err)
	}

	for i := 2; i >= 0; i-- {
		var pop *message.TaskMsg
		pop, err = s.Pop()
		assert.NoError(t, err)
		assert.Equal(t, strconv.Itoa(i), pop.ID)
	}
}
