package store

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"scroll-tech/go-roller/message"

	"github.com/stretchr/testify/assert"
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
		trace := &message.BlockTraces{
			ID:     uint64(i),
			Traces: nil,
		}
		err := s.Push(trace)
		assert.NoError(t, err)
	}

	for i := 2; i >= 0; i-- {
		trace, err := s.Pop()
		assert.NoError(t, err)
		assert.Equal(t, uint64(i), trace.ID)
	}
}
