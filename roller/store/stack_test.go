package store

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"scroll-tech/common/message"

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
		trace := &ProvingTraces{
			Traces: &message.BlockTraces{
				ID:     uint64(i),
				Traces: nil,
			},
		}

		err = s.Push(trace)
		assert.NoError(t, err)
	}

	for i := 2; i >= 0; i-- {
		var pop *ProvingTraces
		pop, err = s.Pop()
		assert.NoError(t, err)
		assert.Equal(t, uint64(i), pop.Traces.ID)
	}

	trace := &ProvingTraces{
		Traces: &message.BlockTraces{
			ID:     1,
			Traces: nil,
		},
	}
	err = s.Push(trace)
	assert.NoError(t, err)
	peak, err := s.Peak()
	assert.NoError(t, err)
	peak2, err := s.Peak()
	assert.NoError(t, err)
	assert.Equal(t, peak, peak2)
}
