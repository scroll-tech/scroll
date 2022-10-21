package store

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"scroll-tech/common/message"
	"testing"

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
			Times: 0,
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

	// test times
	trace := &ProvingTraces{
		Traces: &message.BlockTraces{
			ID:     1,
			Traces: nil,
		},
		Times: 0,
	}
	err = s.Push(trace)
	assert.NoError(t, err)
	pop, err := s.Pop()
	assert.NoError(t, err)
	err = s.Push(pop)
	assert.NoError(t, err)

	pop2, err := s.Pop()
	assert.NoError(t, err)
	assert.Equal(t, 2, pop2.Times)
}
